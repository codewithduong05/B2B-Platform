import { resolveRequestContext } from '../utils/request-context'

export interface HealthResponse {
  status: 'ok'
  requestId: string
}

export default defineEventHandler((event: any) => {
  const requestId = resolveRequestContext(event)
  return { status: 'ok', requestId } satisfies HealthResponse
})
