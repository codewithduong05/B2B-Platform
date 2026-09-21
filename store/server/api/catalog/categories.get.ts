import { resolveUpstreamConfig } from '../../utils/config'
import { request } from '../../utils/upstream-client'

interface UpstreamCategory {
  code: string
  name: string
  slug: string
  description: string
  parent_id: number | null
  sort_order: number
  is_active: boolean
}

interface UpstreamCategoryListResponse {
  items: UpstreamCategory[]
  page: number
  page_size: number
  total: number
  has_next: boolean
}

export default defineEventHandler(async (event) => {
  const query = getQuery(event)
  const page = Number(query.page) || 1
  const pageSize = Number(query.page_size) || 50

  const config = resolveUpstreamConfig()
  const requestId = getHeader(event, 'x-request-id') || crypto.randomUUID()

  const params = new URLSearchParams()
  params.set('page', String(page))
  params.set('page_size', String(pageSize))

  const upstreamPath = `/api/v1/catalog/categories?${params.toString()}`

  try {
    const result = await request(
      {
        baseUrl: config.baseUrl,
        timeoutMs: config.timeoutMs,
        requestId,
      },
      'GET',
      upstreamPath,
    )

    if (result.status !== 200) {
      throw createError({
        statusCode: result.status,
        message: `Upstream categories returned ${result.status}`,
      })
    }

    const data = result.body as UpstreamCategoryListResponse
    return {
      items: data.items || [],
      total: data.total || 0,
      page: data.page || page,
      pageSize: data.page_size || pageSize,
      hasNext: data.has_next || false,
    }
  } catch (error) {
    if (error && typeof error === 'object' && 'statusCode' in error) {
      throw error
    }
    throw createError({
      statusCode: 502,
      message: 'Failed to reach catalog service',
    })
  }
})
