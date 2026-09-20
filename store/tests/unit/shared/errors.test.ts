import { describe, it, expect } from 'vitest'
import {
  createAppError,
  isWireError,
  isAppError,
  toWireError,
  statusCodeToErrorCode,
  normalizeUpstreamError,
} from '#shared/errors'

describe('createAppError', () => {
  it('creates an AppError with all fields', () => {
    const error = createAppError({
      code: 'unauthorized',
      message: 'Active session is required.',
      requestId: 'req_abc123',
      fields: { email: 'Required' },
    })
    expect(error).toEqual({
      code: 'unauthorized',
      message: 'Active session is required.',
      requestId: 'req_abc123',
      fields: { email: 'Required' },
    })
  })

  it('creates an AppError without fields', () => {
    const error = createAppError({
      code: 'not_found',
      message: 'Resource not found',
      requestId: 'req_xyz',
    })
    expect(error).toEqual({
      code: 'not_found',
      message: 'Resource not found',
      requestId: 'req_xyz',
    })
    expect(error.fields).toBeUndefined()
  })

  it('does not include fields when input.fields is undefined', () => {
    const error = createAppError({
      code: 'bad_request',
      message: 'Invalid input',
      requestId: 'req_1',
    })
    expect('fields' in error).toBe(false)
  })
})

describe('isWireError', () => {
  it('returns true for valid wire error', () => {
    expect(
      isWireError({
        detail: 'Something failed',
        code: 'upstream_error',
        request_id: 'req_123',
      }),
    ).toBe(true)
  })

  it('returns false for missing detail', () => {
    expect(isWireError({ code: 'x', request_id: 'y' })).toBe(false)
  })

  it('returns false for missing code', () => {
    expect(isWireError({ detail: 'x', request_id: 'y' })).toBe(false)
  })

  it('returns false for missing request_id', () => {
    expect(isWireError({ detail: 'x', code: 'y' })).toBe(false)
  })

  it('returns false for non-object', () => {
    expect(isWireError('string')).toBe(false)
    expect(isWireError(null)).toBe(false)
    expect(isWireError(42)).toBe(false)
  })
})

describe('isAppError', () => {
  it('returns true for valid AppError', () => {
    expect(
      isAppError({
        code: 'unauthorized',
        message: 'Session required',
        requestId: 'req_1',
      }),
    ).toBe(true)
  })

  it('returns true for AppError with fields', () => {
    expect(
      isAppError({
        code: 'unprocessable_entity',
        message: 'Validation failed',
        requestId: 'req_2',
        fields: { name: 'Required' },
      }),
    ).toBe(true)
  })

  it('returns false for missing code', () => {
    expect(isAppError({ message: 'x', requestId: 'y' })).toBe(false)
  })

  it('returns false for missing message', () => {
    expect(isAppError({ code: 'x', requestId: 'y' })).toBe(false)
  })

  it('returns false for missing requestId', () => {
    expect(isAppError({ code: 'x', message: 'y' })).toBe(false)
  })

  it('returns false for non-object', () => {
    expect(isAppError(null)).toBe(false)
    expect(isAppError(42)).toBe(false)
  })
})

describe('toWireError', () => {
  it('maps AppError to WireError', () => {
    const appError = createAppError({
      code: 'unauthorized',
      message: 'Active session is required.',
      requestId: 'req_abc',
    })
    expect(toWireError(appError)).toEqual({
      detail: 'Active session is required.',
      code: 'unauthorized',
      request_id: 'req_abc',
    })
  })
})

describe('statusCodeToErrorCode', () => {
  it('maps known status codes', () => {
    expect(statusCodeToErrorCode(400)).toBe('bad_request')
    expect(statusCodeToErrorCode(401)).toBe('unauthorized')
    expect(statusCodeToErrorCode(403)).toBe('forbidden')
    expect(statusCodeToErrorCode(404)).toBe('not_found')
    expect(statusCodeToErrorCode(409)).toBe('conflict')
    expect(statusCodeToErrorCode(422)).toBe('unprocessable_entity')
    expect(statusCodeToErrorCode(429)).toBe('rate_limited')
  })

  it('returns upstream_error for unknown codes', () => {
    expect(statusCodeToErrorCode(500)).toBe('upstream_error')
    expect(statusCodeToErrorCode(502)).toBe('upstream_error')
  })
})

describe('normalizeUpstreamError', () => {
  it('passes through valid wire error body', () => {
    const result = normalizeUpstreamError(
      400,
      { detail: 'Bad request', code: 'bad_request', request_id: 'upstream_1' },
      'req_local',
    )
    expect(result).toEqual({
      code: 'bad_request',
      message: 'Bad request',
      requestId: 'upstream_1',
    })
  })

  it('preserves whitespace in wire error fields', () => {
    const result = normalizeUpstreamError(
      401,
      { detail: '  Unauthorized  ', code: '  unauthorized  ', request_id: '  up_2  ' },
      'req_local',
    )
    expect(result).toEqual({
      code: '  unauthorized  ',
      message: '  Unauthorized  ',
      requestId: '  up_2  ',
    })
  })

  it('falls back to status code mapping when wire body has empty trimmed values', () => {
    const result = normalizeUpstreamError(
      404,
      { detail: '   ', code: '   ', request_id: '   ' },
      'req_local',
    )
    expect(result).toEqual({
      code: 'not_found',
      message: 'Upstream request failed with status 404',
      requestId: 'req_local',
    })
  })

  it('maps non-wire error by status code', () => {
    const result = normalizeUpstreamError(404, { error: 'not found' }, 'req_local')
    expect(result).toEqual({
      code: 'not_found',
      message: 'Upstream request failed with status 404',
      requestId: 'req_local',
    })
  })

  it('uses default upstream_error for unknown status', () => {
    const result = normalizeUpstreamError(503, 'service unavailable', 'req_local')
    expect(result).toEqual({
      code: 'upstream_error',
      message: 'Upstream request failed with status 503',
      requestId: 'req_local',
    })
  })
})
