export const SESSION_COOKIE_NAME = 'atlas_session'

export function readSessionCookieFromHeader(
  cookieHeader: string | undefined,
  name: string,
): string | null {
  if (cookieHeader === undefined || cookieHeader.trim() === '') {
    return null
  }
  for (const pair of cookieHeader.split(';')) {
    const trimmed = pair.trim()
    if (trimmed === '') {
      continue
    }
    const separator = trimmed.indexOf('=')
    if (separator <= 0) {
      continue
    }
    const key = trimmed.slice(0, separator).trim()
    if (key !== name) {
      continue
    }
    const rawValue = trimmed.slice(separator + 1).trim()
    try {
      return decodeURIComponent(rawValue)
    } catch {
      return null
    }
  }
  return null
}

export type SessionParser = (sessionToken: string) => unknown

export function resolveSession(cookieHeader: string | undefined, parser: SessionParser): unknown {
  const sessionToken = readSessionCookieFromHeader(cookieHeader, SESSION_COOKIE_NAME)
  if (sessionToken === null) {
    return null
  }
  return parser(sessionToken)
}
