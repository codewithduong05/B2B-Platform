import type { CatalogProduct } from '~~/shared/catalog'
import { resolveUpstreamConfig } from '../utils/config'
import { request } from '../utils/upstream-client'

interface UpstreamProductSummary {
  id: number
  code: string
  slug: string
  sku?: string
  name: string
  short_description: string
  category_code: string
  brand_code: string
  handling_class: string
  base_unit_code: string
  supplier_code: string
  supplier_name: string
  status: string
  is_active: boolean
  is_featured: boolean
  base_price_minor: number | null
  currency: string
  track_inventory: boolean
  created_at: string
  updated_at: string
  published_at: string
}

interface UpstreamProductListResponse {
  items: UpstreamProductSummary[]
  page: number
  page_size: number
  total: number
  has_next: boolean
  facets: Array<{
    facet_type: string
    value_id: string
    value_name: string
    count: number
  }>
}

interface StockLevelSummary {
  id: number
  code: string
  product_id: number
  supplier_id: number
  available_quantity: number
  reserved_quantity: number
  total_quantity: number
  safety_stock: number
}

interface PricingTier {
  min_quantity: number
  max_quantity?: number
  price_minor: number
  currency: string
}

interface PricingTiersResponse {
  product_id: number
  unit_id: number
  base_price_minor: number
  currency: string
  tiers: PricingTier[]
}

function mapProduct(
  item: UpstreamProductSummary,
  stockByProductId: Map<number, StockLevelSummary>,
  pricingByProductId: Map<number, PricingTiersResponse>,
): CatalogProduct {
  const priceMinor = item.base_price_minor ?? 0
  const stock = stockByProductId.get(item.id)
  const available = stock?.available_quantity ?? 0
  const total = stock?.total_quantity ?? 0
  const safetyStock = stock?.safety_stock ?? 0
  const isLowStock = stock != null && available > 0 && available <= safetyStock
  const isOutOfStock = stock == null || total === 0

  let stockLabel = 'Contact for availability'
  if (isOutOfStock) {
    stockLabel = 'Out of stock'
  } else if (isLowStock) {
    stockLabel = `${available} units (low stock)`
  } else if (available > 0) {
    stockLabel = `${available} units available`
  }

  // Find best bulk price from pricing tiers
  let bulkPrice: number | undefined
  let bulkLabel: string | undefined
  const pricing = pricingByProductId.get(item.id)
  if (pricing && pricing.tiers.length > 1) {
    const bestTier = pricing.tiers[pricing.tiers.length - 1]!
    if (bestTier.price_minor < priceMinor) {
      bulkPrice = bestTier.price_minor / 100
      bulkLabel = bestTier.max_quantity
        ? `${bestTier.min_quantity}+ units`
        : `${bestTier.min_quantity}+ units`
    }
  }

  return {
    id: item.slug,
    sku: item.sku || item.code,
    name: item.name,
    description: item.short_description || '',
    supplier: item.supplier_name || item.supplier_code || '',
    price: priceMinor / 100,
    unit: item.base_unit_code || 'unit',
    stock: available,
    stockLabel,
    warehouse: '',
    leadTime: '',
    specs: item.handling_class || '',
    lowStock: isLowStock,
    moq: 1,
    bulkPrice,
    bulkLabel,
  }
}

export default defineEventHandler(async (event) => {
  const query = getQuery(event)
  const page = Number(query.page) || 1
  const pageSize = Number(query.perPage) || 24
  const q = query.q ? String(query.q) : undefined
  const category = query.category ? String(query.category) : undefined
  const brand = query.brand ? String(query.brand) : undefined
  const suppliersRaw = query.suppliers ? String(query.suppliers) : undefined
  const supplier = suppliersRaw ? suppliersRaw.split(',')[0] : undefined
  const sort = query.sortBy ? String(query.sortBy) : undefined

  const config = resolveUpstreamConfig()
  const requestId = getHeader(event, 'x-request-id') || crypto.randomUUID()

  const params = new URLSearchParams()
  params.set('page', String(page))
  params.set('page_size', String(pageSize))
  if (q) params.set('q', q)
  if (category) params.set('category', category)
  if (brand) params.set('brand', brand)
  if (supplier) params.set('supplier', supplier)
  if (sort) params.set('sort', sort)

  const upstreamPath = `/api/v1/catalog/products?${params.toString()}`

  try {
    const result = await request(
      {
        baseUrl: config.baseUrl,
        timeoutMs: config.timeoutMs,
        requestId,
      },
      'GET',
      upstreamPath,
    )

    if (result.status !== 200) {
      throw createError({
        statusCode: result.status,
        message: `Upstream catalog returned ${result.status}`,
      })
    }

    const data = result.body as UpstreamProductListResponse
    const items = data.items || []

    const stockByProductId = new Map<number, StockLevelSummary>()
    if (items.length > 0) {
      const productIds = items.map((p) => p.id).join(',')
      try {
        const stockResult = await request(
          {
            baseUrl: config.baseUrl,
            timeoutMs: config.timeoutMs,
            requestId,
          },
          'GET',
          `/api/v1/inventory/availability?product_ids=${productIds}`,
        )
        if (stockResult.status === 200 && Array.isArray(stockResult.body)) {
          for (const level of stockResult.body as StockLevelSummary[]) {
            const existing = stockByProductId.get(level.product_id)
            if (!existing || level.available_quantity > existing.available_quantity) {
              stockByProductId.set(level.product_id, level)
            }
          }
        }
      } catch {
        // Inventory unavailable — render products without stock data
      }
    }

    const pricingByProductId = new Map<number, PricingTiersResponse>()
    for (const item of items) {
      if (!item.base_price_minor) continue
      try {
        const pricingResult = await request(
          {
            baseUrl: config.baseUrl,
            timeoutMs: config.timeoutMs,
            requestId,
          },
          'GET',
          `/api/v1/pricing/tiers?product_id=${item.id}`,
        )
        if (pricingResult.status === 200) {
          pricingByProductId.set(item.id, pricingResult.body as PricingTiersResponse)
        }
      } catch {
        // Pricing unavailable — use base price only
      }
    }

    const products = items.map((item) => mapProduct(item, stockByProductId, pricingByProductId))

    return {
      products,
      total: data.total || 0,
      page: data.page || page,
      perPage: data.page_size || pageSize,
      hasNext: data.has_next || false,
      facets: data.facets || [],
    }
  } catch (error) {
    if (error && typeof error === 'object' && 'statusCode' in error) {
      throw error
    }
    throw createError({
      statusCode: 502,
      message: 'Failed to reach catalog service',
    })
  }
})
