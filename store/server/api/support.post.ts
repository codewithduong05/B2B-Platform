import { resolveUpstreamConfig } from '../utils/config'
import { request } from '../utils/upstream-client'

export default defineEventHandler(async (event) => {
  const body = await readBody(event)
  const config = resolveUpstreamConfig()
  const requestId = crypto.randomUUID()

  const result = await request(
    {
      baseUrl: config.baseUrl,
      timeoutMs: config.timeoutMs,
      requestId,
    },
    'POST',
    '/api/v1/cms/enquiries',
    body,
  )

  if (result.status !== 201 && result.status !== 200) {
    const detail = (result.body as any)?.detail || 'Failed to submit enquiry'
    throw createError({ statusCode: result.status, message: detail })
  }

  return result.body
})
