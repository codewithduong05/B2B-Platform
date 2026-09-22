import { resolveUpstreamConfig } from '../../utils/config'
import { request } from '../../utils/upstream-client'
import { SESSION_COOKIE_NAME } from '../../utils/session'

export default defineEventHandler(async (event) => {
  const config = resolveUpstreamConfig()
  const requestId = crypto.randomUUID()
  const refreshToken = getCookie(event, 'atlas_refresh')

  if (refreshToken) {
    try {
      await request(
        {
          baseUrl: config.baseUrl,
          timeoutMs: config.timeoutMs,
          requestId,
        },
        'POST',
        '/api/v1/auth/logout',
        { refresh_token: refreshToken },
      )
    } catch {
      // Best-effort revocation — clear cookies regardless
    }
  }

  setCookie(event, SESSION_COOKIE_NAME, '', {
    httpOnly: true,
    secure: true,
    sameSite: 'lax',
    path: '/',
    maxAge: 0,
  })
  setCookie(event, 'atlas_refresh', '', {
    httpOnly: true,
    secure: true,
    sameSite: 'lax',
    path: '/',
    maxAge: 0,
  })

  return { ok: true }
})
