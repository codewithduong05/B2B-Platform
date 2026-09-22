export interface AppError {
  code: string
  message: string
  requestId: string
  fields?: Record<string, string>
}

export interface WireError {
  detail: string
  code: string
  request_id: string
}

export function createAppError(input: AppError): AppError {
  const error: AppError = {
    code: input.code,
    message: input.message,
    requestId: input.requestId,
  }
  if (input.fields !== undefined) {
    error.fields = input.fields
  }
  return error
}

export function isWireError(value: unknown): value is WireError {
  if (typeof value !== 'object' || value === null) {
    return false
  }
  const candidate = value as Record<string, unknown>
  return (
    typeof candidate.detail === 'string' &&
    typeof candidate.code === 'string' &&
    typeof candidate.request_id === 'string'
  )
}

export function isAppError(value: unknown): value is AppError {
  if (typeof value !== 'object' || value === null) {
    return false
  }
  const candidate = value as Record<string, unknown>
  if (
    typeof candidate.code !== 'string' ||
    typeof candidate.message !== 'string' ||
    typeof candidate.requestId !== 'string'
  ) {
    return false
  }
  if (candidate.fields === undefined) {
    return true
  }
  const fields = candidate.fields
  if (typeof fields !== 'object' || fields === null) {
    return false
  }
  return Object.values(fields).every((fieldValue) => typeof fieldValue === 'string')
}

export function toWireError(error: AppError): WireError {
  return {
    detail: error.message,
    code: error.code,
    request_id: error.requestId,
  }
}

export function statusCodeToErrorCode(status: number): string {
  switch (status) {
    case 400:
      return 'bad_request'
    case 401:
      return 'unauthorized'
    case 403:
      return 'forbidden'
    case 404:
      return 'not_found'
    case 409:
      return 'conflict'
    case 422:
      return 'unprocessable_entity'
    case 429:
      return 'rate_limited'
    default:
      return 'upstream_error'
  }
}

export function normalizeUpstreamError(status: number, body: unknown, requestId: string): AppError {
  const fallbackMessage = `Upstream request failed with status ${status}`
  if (isWireError(body)) {
    const code = body.code.trim() === '' ? statusCodeToErrorCode(status) : body.code
    const message = body.detail.trim() === '' ? fallbackMessage : body.detail
    const wireRequestId = body.request_id.trim() === '' ? requestId : body.request_id
    return createAppError({ code, message, requestId: wireRequestId })
  }
  return createAppError({
    code: statusCodeToErrorCode(status),
    message: fallbackMessage,
    requestId,
  })
}
