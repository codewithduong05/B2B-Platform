import { resolveUpstreamConfig } from '~~/server/utils/config'
import { request } from '~~/server/utils/upstream-client'
import { resolveRequestContext } from '~~/server/utils/request-context'

export default defineEventHandler(async (event) => {
  const config = resolveUpstreamConfig()
  const requestId = resolveRequestContext(event)

  const [products, orders, lowStock] = await Promise.allSettled([
    request({ baseUrl: config.baseUrl, timeoutMs: config.timeoutMs, requestId }, 'GET', '/admin/catalog/products?page_size=1'),
    request({ baseUrl: config.baseUrl, timeoutMs: config.timeoutMs, requestId }, 'GET', '/admin/orders?page_size=1'),
    request({ baseUrl: config.baseUrl, timeoutMs: config.timeoutMs, requestId }, 'GET', '/inventory/low-stock'),
  ])

  const productCount = products.status === 'fulfilled' && products.value.body && typeof products.value.body === 'object'
    ? (products.value.body as { total?: number }).total ?? 0
    : 0

  const orderCount = orders.status === 'fulfilled' && orders.value.body && typeof orders.value.body === 'object'
    ? (orders.value.body as { total?: number }).total ?? 0
    : 0

  const lowStockItems = lowStock.status === 'fulfilled' && lowStock.value.body && typeof lowStock.value.body === 'object'
    ? Array.isArray((lowStock.value.body as { items?: unknown[] }).items)
      ? (lowStock.value.body as { items: unknown[] }).items.length
      : 0
    : 0

  return { productCount, orderCount, lowStockItems }
})
