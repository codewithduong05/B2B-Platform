import { resolveUpstreamConfig } from '../../../utils/config'
import { request } from '../../../utils/upstream-client'

export default defineEventHandler(async (event) => {
  const config = resolveUpstreamConfig()
  const requestId = getHeader(event, 'x-request-id') || crypto.randomUUID()
  const sessionToken = getCookie(event, 'atlas_session') || ''
  const code = getRouterParam(event, 'code')
  const body = await readBody(event)

  if (!code) {
    throw createError({ statusCode: 400, message: 'Missing cart item code' })
  }

  const result = await request(
    {
      baseUrl: config.baseUrl,
      timeoutMs: config.timeoutMs,
      requestId,
      accessToken: sessionToken || undefined,
    },
    'PATCH',
    `/api/v1/commerce/cart/items/${encodeURIComponent(code)}`,
    body,
  )

  if (result.status === 401) {
    throw createError({ statusCode: 401, message: 'Authentication required' })
  }

  if (result.status === 404) {
    throw createError({ statusCode: 404, message: 'Cart item not found' })
  }

  if (result.status !== 200) {
    throw createError({ statusCode: result.status, message: 'Failed to update cart item' })
  }

  return result.body
})
