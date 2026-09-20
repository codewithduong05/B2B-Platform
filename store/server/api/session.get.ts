import { getHeader } from 'h3'
import { createAppError } from '../../shared/errors'
import { verifySessionToken } from '../utils/auth'
import { resolveRequestContext } from '../utils/request-context'
import { sendAppError } from '../utils/errors'
import { readSessionCookieFromHeader, SESSION_COOKIE_NAME } from '../utils/session'

export default defineEventHandler(async (event: any) => {
  const requestId = resolveRequestContext(event)
  const cookieHeader = getHeader(event, 'cookie')
  const sessionToken = readSessionCookieFromHeader(cookieHeader, SESSION_COOKIE_NAME)

  if (sessionToken === null) {
    return sendAppError(
      event,
      createAppError({ code: 'unauthorized', message: 'Active session is required.', requestId }),
      401,
    )
  }

  const principal = await verifySessionToken(event, sessionToken)
  if (principal === null) {
    return sendAppError(
      event,
      createAppError({
        code: 'unauthorized',
        message: 'Session could not be verified.',
        requestId,
      }),
      401,
    )
  }

  return principal
})
