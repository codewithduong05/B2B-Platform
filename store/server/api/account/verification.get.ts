import { resolveUpstreamConfig } from '../../utils/config'
import { request } from '../../utils/upstream-client'

export default defineEventHandler(async (event) => {
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
    '/api/v1/buyer/me/verification',
  )

  if (result.status === 404) {
    return { status: 'not_submitted' }
  }

  if (result.status !== 200) {
    throw createError({ statusCode: result.status, message: 'Failed to fetch verification' })
  }

  return result.body
})
