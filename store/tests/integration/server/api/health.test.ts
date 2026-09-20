import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.hoisted(() => {
  vi.stubGlobal('defineEventHandler', (fn: any) => fn)
})

vi.mock('h3', () => ({
  setHeader: vi.fn(),
}))

vi.mock('#server/utils/request-context', () => ({
  resolveRequestContext: vi.fn((event: any) => {
    const existing = event.context?.requestId
    if (existing) return existing
    const id = 'req_health_test'
    event.context = event.context || {}
    event.context.requestId = id
    return id
  }),
}))

import handler from '#server/api/health.get'

function createMockEvent(): any {
  return {
    node: { req: { headers: {} } },
    context: {},
  }
}

describe('health.get handler', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns status ok with requestId', () => {
    const event = createMockEvent()
    const result = handler(event)
    expect(result).toEqual({
      status: 'ok',
      requestId: 'req_health_test',
    })
  })

  it('returns consistent requestId across calls', () => {
    const event = createMockEvent()
    const result1 = handler(event)
    const result2 = handler(event)
    expect(result1.requestId).toBe(result2.requestId)
  })
})
