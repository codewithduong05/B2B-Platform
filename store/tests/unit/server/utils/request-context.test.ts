import { describe, it, expect } from 'vitest'
import { REQUEST_ID_HEADER, createFallbackRequestId } from '#server/utils/request-context'

describe('REQUEST_ID_HEADER', () => {
  it('is x-request-id', () => {
    expect(REQUEST_ID_HEADER).toBe('x-request-id')
  })
})

describe('createFallbackRequestId', () => {
  it('returns a string starting with req_', () => {
    const id = createFallbackRequestId()
    expect(id).toMatch(/^req_/)
  })

  it('returns unique values', () => {
    const id1 = createFallbackRequestId()
    const id2 = createFallbackRequestId()
    expect(id1).not.toBe(id2)
  })
})
