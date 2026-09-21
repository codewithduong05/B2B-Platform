import { resolveUpstreamConfig } from '../../utils/config'
import { request } from '../../utils/upstream-client'

interface UpstreamSupplier {
  code: string
  name: string
  slug: string
  is_active: boolean
}

export default defineEventHandler(async () => {
  const config = resolveUpstreamConfig()
  const requestId = crypto.randomUUID()

  let suppliers: UpstreamSupplier[] = []
  try {
    const result = await request(
      {
        baseUrl: config.baseUrl,
        timeoutMs: config.timeoutMs,
        requestId,
      },
      'GET',
      '/api/v1/catalog/suppliers?page_size=50',
    )
    if (result.status === 200) {
      const data = result.body as { items: UpstreamSupplier[] }
      suppliers = (data.items || []).filter((s) => s.is_active)
    }
  } catch {
    // Suppliers unavailable
  }

  return {
    suppliers: suppliers.map((s) => ({
      name: s.name,
      code: s.code,
      slug: s.slug,
    })),
  }
})
