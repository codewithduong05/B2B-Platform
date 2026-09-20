import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.hoisted(() => {
  vi.stubGlobal('defineEventHandler', (fn: any) => fn)
})

vi.mock('h3', () => ({
  getHeader: vi.fn((event: any, name: string) => {
    return (event.node?.req?.headers?.[name] as string) ?? null
  }),
  createError: vi.fn((opts: any) => {
    const err = new Error(opts?.statusMessage)
    Object.assign(err, opts)
    return err
  }),
  setHeader: vi.fn(),
}))

vi.mock('#server/utils/request-context', () => ({
  resolveRequestContext: vi.fn((event: any) => {
    const existing = event.context?.requestId
    if (existing) return existing
    const id = 'req_mock_123'
    event.context = event.context || {}
    event.context.requestId = id
    return id
  }),
}))

vi.mock('#server/utils/auth', () => ({
  verifySessionToken: vi.fn(),
}))

vi.mock('#server/utils/errors', () => ({
  sendAppError: vi.fn((_event: any, error: any, statusCode: number) => {
    const err = new Error(error?.message)
    Object.assign(err, { statusCode, data: error })
    return err
  }),
}))

vi.mock('#server/utils/session', () => ({
  SESSION_COOKIE_NAME: 'atlas_session',
  readSessionCookieFromHeader: vi.fn((header: string | undefined, _name: string) => {
    if (!header) return null
    const match = header.match(/atlas_session=([^;]+)/)
    return match?.[1] ?? null
  }),
}))

import handler from '#server/api/session.get'
import { verifySessionToken } from '#server/utils/auth'
import { sendAppError } from '#server/utils/errors'
import { readSessionCookieFromHeader } from '#server/utils/session'

function createMockEvent(headers: Record<string, string> = {}): any {
  return {
    node: { req: { headers } },
    context: {},
  }
}

describe('session.get handler', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns 401 when no session cookie present', async () => {
    const event = createMockEvent({})
    vi.mocked(readSessionCookieFromHeader).mockReturnValue(null)
    vi.mocked(sendAppError).mockImplementation((() => {
      throw new Error('Active session is required.')
    }) as never)

    await expect(handler(event)).rejects.toThrow('Active session is required.')
    expect(sendAppError).toHaveBeenCalledOnce()
  })

  it('returns 401 when session cannot be verified', async () => {
    const event = createMockEvent({ cookie: 'atlas_session=tok_invalid' })
    vi.mocked(readSessionCookieFromHeader).mockReturnValue('tok_invalid')
    vi.mocked(verifySessionToken).mockResolvedValue(null)
    vi.mocked(sendAppError).mockImplementation((() => {
      throw new Error('Session could not be verified.')
    }) as never)

    await expect(handler(event)).rejects.toThrow('Session could not be verified.')
    expect(verifySessionToken).toHaveBeenCalledWith(event, 'tok_invalid')
  })

  it('returns principal when session is valid', async () => {
    const event = createMockEvent({ cookie: 'atlas_session=tok_valid' })
    vi.mocked(readSessionCookieFromHeader).mockReturnValue('tok_valid')
    const principal = { id: 'user_1', permissions: ['admin:health'] }
    vi.mocked(verifySessionToken).mockResolvedValue(principal)

    const result = await handler(event)
    expect(result).toEqual(principal)
  })
})
