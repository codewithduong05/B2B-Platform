import { resolveUpstreamConfig } from '~~/server/utils/config'
import { request } from '~~/server/utils/upstream-client'
import { resolveRequestContext } from '~~/server/utils/request-context'
import { normalizeUpstreamError } from '~~/shared/errors'

export default defineEventHandler(async (event) => {
  const config = resolveUpstreamConfig()
  const requestId = resolveRequestContext(event)
  const id = getRouterParam(event, 'id')

  const result = await request({
    baseUrl: config.baseUrl,
    timeoutMs: config.timeoutMs,
    requestId,
  }, 'GET', `/admin/orders/${id}`)

  if (result.status >= 400) {
    throw normalizeUpstreamError(result.status, result.body, requestId)
  }

  return result.body
})
