import { resolveUpstreamConfig } from '../../utils/config'
import { request } from '../../utils/upstream-client'

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
    },
    'POST',
    '/api/v1/buyer/me/addresses',
    body,
  )

  if (result.status !== 201 && result.status !== 200) {
    const detail = (result.body as any)?.detail || 'Failed to create address'
    throw createError({ statusCode: result.status, message: detail })
  }

  return result.body
})
