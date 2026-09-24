import { resolveUpstreamConfig } from '../../utils/config'
import { request } from '../../utils/upstream-client'

interface ApiCategory {
  code: string
  name: string
  slug: string
  description: string
  sort_order: number
  is_active: boolean
}

interface UpstreamListResponse {
  total: number
}

export default defineEventHandler(async () => {
  const config = resolveUpstreamConfig()
  const requestId = crypto.randomUUID()

  let categories: ApiCategory[] = []
  try {
    const result = await request(
      {
        baseUrl: config.baseUrl,
        timeoutMs: config.timeoutMs,
        requestId,
      },
      'GET',
      '/api/v1/catalog/categories?page=1&page_size=50',
    )
    if (result.status === 200) {
      const data = result.body as { items: ApiCategory[] }
      categories = (data.items || []).filter((c) => c.is_active)
    }
  } catch {
    // Categories unavailable
  }

  // Fetch product counts per category
  const categoriesWithCounts = await Promise.all(
    categories.map(async (cat) => {
      let count = 0
      try {
        const result = await request(
          {
            baseUrl: config.baseUrl,
            timeoutMs: config.timeoutMs,
            requestId,
          },
          'GET',
          `/api/v1/catalog/products?page=1&page_size=1&category=${cat.code}`,
        )
        if (result.status === 200) {
          count = (result.body as UpstreamListResponse).total || 0
        }
      } catch {
        // ignore
      }
      return { ...cat, productCount: count }
    }),
  )

  return { items: categoriesWithCounts }
})
