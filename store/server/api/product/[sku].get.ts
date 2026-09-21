import type { ProductDetail } from '~~/shared/product'
import { resolveUpstreamConfig } from '../../utils/config'
import { request } from '../../utils/upstream-client'

interface UpstreamProductDetail {
  code: string
  slug: string
  name: string
  short_description: string
  description: string
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
  weight_grams: number | null
  length_mm: number | null
  width_mm: number | null
  height_mm: number | null
  gtin: string | null
  sku: string | null
  category: {
    code: string
    name: string
    slug: string
    description: string
    parent_code: string | null
    sort_order: number
    is_active: boolean
  } | null
  brand: {
    code: string
    name: string
    slug: string
    description: string
    logo_url: string | null
    website_url: string | null
    is_active: boolean
  } | null
  base_unit: {
    code: string
    name: string
    symbol: string
    unit_type: string
    base_unit_code: string
    conversion_factor: number
    is_active: boolean
  } | null
  supplier: {
    code: string
    supplier_id: number
    name: string
    slug: string
    description: string
    logo_url: string | null
    is_active: boolean
  } | null
  units: Array<{
    code: string
    unit_code: string
    unit_name: string
    unit_symbol: string
    conversion_factor: number
    is_default: boolean
    price_minor: number
  }> | null
  media: Array<{
    code: string
    url: string
    alt_text: string
    media_type: string
    sort_order: number
    is_primary: boolean
    width_px: number | null
    height_px: number | null
    file_size_bytes: number | null
    mime_type: string | null
  }> | null
  attributes: Array<{
    attribute_id: string
    attribute_name: string
    attribute_type: string
    value_id: string
    value_name: string
    text_value: string
    number_value: number | null
    boolean_value: boolean | null
  }> | null
}

interface UpstreamProductDetailResponse {
  product: UpstreamProductDetail
}

function mapProductDetail(data: UpstreamProductDetail): ProductDetail {
  const priceMinor = data.base_price_minor ?? 0
  const unitPrice = priceMinor / 100
  const unitName = data.base_unit?.name || data.base_unit_code || 'unit'

  const specs = [
    { label: 'Product Code', value: data.code },
    { label: 'SKU', value: data.sku || data.code },
    { label: 'Category', value: data.category?.name || data.category_code || 'N/A' },
    { label: 'Brand', value: data.brand?.name || data.brand_code || 'N/A' },
    { label: 'Handling Class', value: data.handling_class || 'N/A' },
    { label: 'Unit', value: `${unitName} (${data.base_unit?.symbol || ''})` },
  ]

  if (data.weight_grams) {
    specs.push({ label: 'Weight', value: `${data.weight_grams}g` })
  }
  if (data.length_mm && data.width_mm && data.height_mm) {
    specs.push({
      label: 'Dimensions',
      value: `${data.length_mm} × ${data.width_mm} × ${data.height_mm} mm`,
    })
  }
  if (data.gtin) {
    specs.push({ label: 'GTIN', value: data.gtin })
  }

  if (data.attributes) {
    for (const attr of data.attributes) {
      specs.push({
        label: attr.attribute_name,
        value: attr.value_name || attr.text_value || String(attr.number_value || ''),
      })
    }
  }

  const images = (data.media || [])
    .filter((m) => m.media_type === 'image')
    .sort((a, b) => a.sort_order - b.sort_order)
    .map((m) => ({
      label: m.alt_text || `Image ${m.sort_order + 1}`,
      alt: m.alt_text || data.name,
      src: m.url,
      active: m.is_primary,
    }))

  if (images.length === 0) {
    images.push({ label: 'No image', alt: data.name, src: '', active: true })
  }

  const unitSymbol = data.base_unit?.symbol || unitName

  const tiers = [
    { id: 'tier-1', range: '1 - 9', price: unitPrice, discount: 'Standard' },
  ]

  return {
    id: data.code,
    sku: data.sku || data.code,
    name: data.name,
    description: data.description || data.short_description || '',
    mpn: data.sku || data.code,
    supplier: {
      name: data.supplier?.name || data.supplier_code || 'Unknown',
      id: data.supplier?.code || data.supplier_code || '',
      tier: '',
      audited: false,
      onTimeSla: '',
      qualityDefect: '',
      terms: '',
      compliance: [],
    },
    pricing: {
      unitPrice,
      unit: unitSymbol,
      tiers,
    },
    stock: [],
    totalStock: 0,
    specs,
    images,
    artifacts: [],
    category: [data.category?.name || data.category_code || 'Products'],
    unspsc: data.category_code || '',
    certifications: data.is_featured ? ['Featured Product'] : [],
  }
}

export default defineEventHandler(async (event) => {
  const sku = getRouterParam(event, 'sku')
  if (!sku) {
    throw createError({ statusCode: 400, message: 'Missing product SKU' })
  }

  const config = resolveUpstreamConfig()
  const requestId = getHeader(event, 'x-request-id') || crypto.randomUUID()

  const upstreamPath = `/api/v1/catalog/products/${encodeURIComponent(sku)}`

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

    if (result.status === 404) {
      throw createError({ statusCode: 404, message: 'Product not found' })
    }

    if (result.status !== 200) {
      throw createError({
        statusCode: result.status,
        message: `Upstream catalog returned ${result.status}`,
      })
    }

    const data = result.body as UpstreamProductDetailResponse
    return mapProductDetail(data.product)
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
