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
    'PATCH',
    '/api/v1/buyer/me/',
    body,
  )

  if (result.status !== 200) {
    const detail = (result.body as any)?.detail || 'Failed to update profile'
    throw createError({ statusCode: result.status, message: detail })
  }

  return result.body
})
