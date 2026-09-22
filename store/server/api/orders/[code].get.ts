import { resolveUpstreamConfig } from '../../utils/config'
import { request } from '../../utils/upstream-client'

export default defineEventHandler(async (event) => {
  const code = getRouterParam(event, 'code')
  if (!code) {
    throw createError({ statusCode: 400, message: 'Order code is required' })
  }

  const config = resolveUpstreamConfig()
  const requestId = crypto.randomUUID()
  const sessionToken = getCookie(event, 'atlas_session') || ''

  const result = await request(
    {
      baseUrl: config.baseUrl,
      timeoutMs: config.timeoutMs,
      requestId,
      accessToken: sessionToken || undefined,
    },
    'GET',
    `/api/v1/commerce/orders/me/${code}`,
  )

  if (result.status === 401) {
    throw createError({ statusCode: 401, message: 'Authentication required' })
  }

  if (result.status === 404) {
    throw createError({ statusCode: 404, message: 'Order not found' })
  }

  if (result.status !== 200) {
    const detail = (result.body as any)?.detail || 'Failed to fetch order'
    throw createError({ statusCode: result.status, message: detail })
  }

  return result.body
})
