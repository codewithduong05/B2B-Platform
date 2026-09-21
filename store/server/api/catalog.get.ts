import type { CatalogProduct } from '~~/shared/catalog'
import { resolveUpstreamConfig } from '../utils/config'
import { request } from '../utils/upstream-client'

interface UpstreamProductSummary {
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

function mapProduct(item: UpstreamProductSummary): CatalogProduct {
  const priceMinor = item.base_price_minor ?? 0
  return {
    id: item.slug,
    sku: item.sku || item.code,
    name: item.name,
    description: item.short_description || '',
    supplier: item.supplier_code || '',
    price: priceMinor / 100,
    unit: item.base_unit_code || 'unit',
    stock: 0,
    stockLabel: 'Contact for availability',
    warehouse: '',
    leadTime: '',
    specs: item.handling_class || '',
    lowStock: false,
    moq: 1,
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
    const products = (data.items || []).map(mapProduct)

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
