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
  if (query.q) params.set('q', String(query.q))
  if (query.category) params.set('category', String(query.category))
  if (query.brand) params.set('brand', String(query.brand))
  if (query.supplier) params.set('supplier', String(query.supplier))
  if (query.sort) params.set('sort', String(query.sort))

  const qs = params.toString()
  const path = `/admin/catalog/products${qs ? `?${qs}` : ''}`

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
