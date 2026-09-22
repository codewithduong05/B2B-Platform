import { resolveUpstreamConfig } from '../../utils/config'
import { request } from '../../utils/upstream-client'
import { SESSION_COOKIE_NAME } from '../../utils/session'

export default defineEventHandler(async (event) => {
  const body = await readBody(event)
  const config = resolveUpstreamConfig()
  const requestId = crypto.randomUUID()

  const email = body?.email
  const password = body?.password

  if (!email || !password) {
    throw createError({ statusCode: 400, message: 'Email and password are required' })
  }

  const result = await request(
    {
      baseUrl: config.baseUrl,
      timeoutMs: config.timeoutMs,
      requestId,
    },
    'POST',
    '/api/v1/auth/login',
    { email, password },
  )

  if (result.status !== 200) {
    const detail = (result.body as any)?.detail || 'Invalid credentials'
    throw createError({ statusCode: result.status, message: detail })
  }

  const data = result.body as {
    access_token: string
    token_type: string
    expires_in: number
    refresh_token?: string
  }

  setCookie(event, SESSION_COOKIE_NAME, data.access_token, {
    httpOnly: true,
    secure: true,
    sameSite: 'lax',
    path: '/',
    maxAge: data.expires_in || 3600,
  })

  if (data.refresh_token) {
    setCookie(event, 'atlas_refresh', data.refresh_token, {
      httpOnly: true,
      secure: true,
      sameSite: 'lax',
      path: '/',
      maxAge: 30 * 24 * 60 * 60,
    })
  }

  return { ok: true }
})
