import { resolveUpstreamConfig } from '../../utils/config'
import { request } from '../../utils/upstream-client'

export default defineEventHandler(async (event) => {
  const config = resolveUpstreamConfig()
  const requestId = getHeader(event, 'x-request-id') || crypto.randomUUID()
  const sessionToken = getCookie(event, 'atlas_session') || ''
  const body = await readBody(event)

  const result = await request(
    {
      baseUrl: config.baseUrl,
      timeoutMs: config.timeoutMs,
      requestId,
      accessToken: sessionToken || undefined,
    },
    'POST',
    '/api/v1/commerce/cart/items',
    body,
  )

  if (result.status === 401) {
    throw createError({ statusCode: 401, message: 'Authentication required' })
  }

  if (result.status === 404) {
    throw createError({ statusCode: 404, message: 'Product not found' })
  }

  if (result.status !== 201 && result.status !== 200) {
    throw createError({ statusCode: result.status, message: 'Failed to add item to cart' })
  }

  return result.body
})
