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
    '/api/v1/buyer/me/addresses',
  )

  if (result.status !== 200) {
    throw createError({ statusCode: result.status, message: 'Failed to fetch addresses' })
  }

  return result.body
})
