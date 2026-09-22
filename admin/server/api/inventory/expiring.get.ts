import { resolveUpstreamConfig } from '~~/server/utils/config'
import { request } from '~~/server/utils/upstream-client'
import { resolveRequestContext } from '~~/server/utils/request-context'
import { normalizeUpstreamError } from '~~/shared/errors'

export default defineEventHandler(async (event) => {
  const config = resolveUpstreamConfig()
  const requestId = resolveRequestContext(event)
  const query = getQuery(event)

  const params = new URLSearchParams()
  if (query.horizon_days) params.set('horizon_days', String(query.horizon_days))

  const qs = params.toString()
  const path = `/inventory/expiring${qs ? `?${qs}` : ''}`

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
