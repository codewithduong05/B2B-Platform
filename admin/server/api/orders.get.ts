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
  if (query.status) params.set('status', String(query.status))
  if (query.buyer_id) params.set('buyer_id', String(query.buyer_id))

  const qs = params.toString()
  const path = `/api/v1/admin/orders${qs ? `?${qs}` : ''}`

  const result = await request({
    baseUrl: config.baseUrl,
    timeoutMs: config.timeoutMs,
    requestId,
    accessToken: getCookie(event, 'atlas_admin_session') || undefined,
  }, 'GET', path)

  if (result.status >= 400) {
    throw normalizeUpstreamError(result.status, result.body, requestId)
  }

  return result.body
})
