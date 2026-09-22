import { resolveUpstreamConfig } from '~~/server/utils/config'
import { request } from '~~/server/utils/upstream-client'
import { resolveRequestContext } from '~~/server/utils/request-context'
import { normalizeUpstreamError } from '~~/shared/errors'

export default defineEventHandler(async (event) => {
  const config = resolveUpstreamConfig()
  const requestId = resolveRequestContext(event)
  const query = getQuery(event)

  const params = new URLSearchParams()
  if (query.page) params.set('page', String(query.page))
  if (query.page_size) params.set('page_size', String(query.page_size))

  const qs = params.toString()
  const path = `/admin/users${qs ? `?${qs}` : ''}`

  const result = await request({
    baseUrl: config.baseUrl,
    timeoutMs: config.timeoutMs,
    requestId,
  }, 'GET', path)

  if (result.status >= 400) {
    throw normalizeUpstreamError(result.status, result.body, requestId)
  }

  return result.body
})
