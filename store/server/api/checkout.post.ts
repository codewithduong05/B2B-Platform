import { resolveUpstreamConfig } from '../utils/config'
import { request } from '../utils/upstream-client'

export default defineEventHandler(async (event) => {
  const body = await readBody(event)
  const config = resolveUpstreamConfig()
  const requestId = crypto.randomUUID()
  const sessionToken = getCookie(event, 'atlas_session') || ''

  const result = await request(
    {
      baseUrl: config.baseUrl,
      timeoutMs: config.timeoutMs,
      requestId,
      accessToken: sessionToken || undefined,
      headers: { 'Idempotency-Key': requestId },
    },
    'POST',
    '/api/v1/commerce/checkout',
    body,
  )

  if (result.status !== 200 && result.status !== 201) {
    const detail = (result.body as any)?.detail || 'Checkout failed'
    throw createError({ statusCode: result.status, message: detail })
  }

  return result.body
})
