import { resolveUpstreamConfig } from '../../utils/config'
import { request } from '../../utils/upstream-client'

export default defineEventHandler(async () => {
  const config = resolveUpstreamConfig()
  const requestId = crypto.randomUUID()

  try {
    const result = await request(
      {
        baseUrl: config.baseUrl,
        timeoutMs: config.timeoutMs,
        requestId,
      },
      'GET',
      '/api/v1/catalog/facets',
    )
    if (result.status === 200) {
      return result.body
    }
    return { facets: [] }
  } catch {
    return { facets: [] }
  }
})
