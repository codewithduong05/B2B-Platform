import { resolveUpstreamConfig } from '~~/server/utils/config'
import { request } from '~~/server/utils/upstream-client'
import { resolveRequestContext } from '~~/server/utils/request-context'
import { getCookie } from 'h3'

interface UpstreamListResponse {
  total: number
}

async function fetchCount(
  config: ReturnType<typeof resolveUpstreamConfig>,
  requestId: string,
  path: string,
  accessToken?: string,
): Promise<number> {
  try {
    const result = await request(
      {
        baseUrl: config.baseUrl,
        timeoutMs: config.timeoutMs,
        requestId,
        accessToken,
      },
      'GET',
      `/api/v1${path}`,
    )
    if (result.status === 200) {
      return (result.body as UpstreamListResponse).total || 0
    }
  } catch {
    // ignore
  }
  return 0
}

export default defineEventHandler(async (event) => {
  const config = resolveUpstreamConfig()
  const requestId = resolveRequestContext(event)
  const accessToken = getCookie(event, 'atlas_admin_session') || undefined

  const [products, orders, lowStock, pendingOrdersCount] = await Promise.allSettled([
    fetchCount(config, requestId, '/admin/catalog/products?page_size=1', accessToken),
    fetchCount(config, requestId, '/admin/orders?page_size=1', accessToken),
    fetchCount(config, requestId, '/inventory/low-stock', accessToken),
    fetchCount(config, requestId, '/admin/orders?page_size=1&status=placed', accessToken), // Assuming 'placed' is a pending status
  ])

  const productCount = products.status === 'fulfilled' ? products.value : 0
  const orderCount = orders.status === 'fulfilled' ? orders.value : 0
  const lowStockItems = lowStock.status === 'fulfilled' ? lowStock.value : 0
  const pendingOrders = pendingOrdersCount.status === 'fulfilled' ? pendingOrdersCount.value : 0

  return { productCount, orderCount, lowStockItems, pendingOrders }
})
