import { describe, it, expect } from 'vitest'
import {
  SESSION_COOKIE_NAME,
  readSessionCookieFromHeader,
  resolveSession,
} from '#server/utils/session'

describe('SESSION_COOKIE_NAME', () => {
  it('is atlas_session', () => {
    expect(SESSION_COOKIE_NAME).toBe('atlas_session')
  })
})

describe('readSessionCookieFromHeader', () => {
  it('returns null for undefined header', () => {
    expect(readSessionCookieFromHeader(undefined, SESSION_COOKIE_NAME)).toBeNull()
  })

  it('returns null for empty string', () => {
    expect(readSessionCookieFromHeader('', SESSION_COOKIE_NAME)).toBeNull()
  })

  it('returns null for whitespace-only string', () => {
    expect(readSessionCookieFromHeader('   ', SESSION_COOKIE_NAME)).toBeNull()
  })

  it('extracts session cookie value', () => {
    const header = 'other=val; atlas_session=tok_abc123; another=val'
    expect(readSessionCookieFromHeader(header, SESSION_COOKIE_NAME)).toBe('tok_abc123')
  })

  it('decodes URI-encoded values', () => {
    const header = 'atlas_session=tok%20abc%2F123'
    expect(readSessionCookieFromHeader(header, SESSION_COOKIE_NAME)).toBe('tok abc/123')
  })

  it('returns null when cookie name not found', () => {
    const header = 'other=val; another=val'
    expect(readSessionCookieFromHeader(header, SESSION_COOKIE_NAME)).toBeNull()
  })

  it('handles cookies without values', () => {
    const header = 'atlas_session; other=val'
    expect(readSessionCookieFromHeader(header, SESSION_COOKIE_NAME)).toBeNull()
  })

  it('handles malformed cookie pairs', () => {
    const header = '; ; atlas_session=tok; ;'
    expect(readSessionCookieFromHeader(header, SESSION_COOKIE_NAME)).toBe('tok')
  })
})

describe('resolveSession', () => {
  it('returns null when no cookie present', () => {
    const parser = (token: string) => ({ token })
    expect(resolveSession(undefined, parser)).toBeNull()
  })

  it('calls parser with session token', () => {
    const parser = (token: string) => ({ parsed: token })
    const result = resolveSession('atlas_session=tok_xyz', parser)
    expect(result).toEqual({ parsed: 'tok_xyz' })
  })

  it('returns parser result directly', () => {
    const parser = (token: string) => (token === 'valid' ? { ok: true } : null)
    expect(resolveSession('atlas_session=valid', parser)).toEqual({ ok: true })
    expect(resolveSession('atlas_session=invalid', parser)).toBeNull()
  })
})
