import { randomUUID } from 'node:crypto'
import { setHeader } from 'h3'

declare module 'h3' {
  interface H3EventContext {
    requestId?: string
  }
}

export const REQUEST_ID_HEADER = 'x-request-id'

const REQUEST_ID_PATTERN = /^[a-zA-Z0-9._-]{1,128}$/

export function createFallbackRequestId(): string {
  return `req_${randomUUID()}`
}

export function resolveRequestContext(event: any): string {
  const existing = event.context.requestId
  if (existing !== undefined && existing !== '') {
    return existing
  }

  const rawHeader = event.node.req.headers[REQUEST_ID_HEADER]
  const provided = Array.isArray(rawHeader) ? rawHeader[0] : rawHeader
  const requestId =
    typeof provided === 'string' && REQUEST_ID_PATTERN.test(provided)
      ? provided
      : createFallbackRequestId()

  event.context.requestId = requestId
  setHeader(event, REQUEST_ID_HEADER, requestId)
  return requestId
}
