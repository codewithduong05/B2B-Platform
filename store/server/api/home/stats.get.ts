import { resolveUpstreamConfig } from '../../utils/config'
import { request } from '../../utils/upstream-client'

interface UpstreamListResponse {
  total: number
}

async function fetchCount(
  config: ReturnType<typeof resolveUpstreamConfig>,
  requestId: string,
  path: string,
): Promise<number> {
  try {
    const result = await request(
      {
        baseUrl: config.baseUrl,
        timeoutMs: config.timeoutMs,
        requestId,
      },
      'GET',
      path,
    )
    if (result.status === 200) {
      return (result.body as UpstreamListResponse).total || 0
    }
  } catch {
    // ignore
  }
  return 0
}

export default defineEventHandler(async () => {
  const config = resolveUpstreamConfig()
  const requestId = crypto.randomUUID()

  const [products, categories, brands, suppliers] = await Promise.all([
    fetchCount(config, requestId, '/api/v1/catalog/products?page=1&page_size=1'),
    fetchCount(config, requestId, '/api/v1/catalog/categories?page=1&page_size=1'),
    fetchCount(config, requestId, '/api/v1/catalog/brands?page=1&page_size=1'),
    fetchCount(config, requestId, '/api/v1/catalog/suppliers?page=1&page_size=1'),
  ])

  return { products, categories, brands, suppliers }
})
