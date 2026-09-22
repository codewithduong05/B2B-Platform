import { createAppError, type AppError } from '../../shared/errors'

export type UpstreamMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export interface UpstreamRequestConfig {
  baseUrl: string
  timeoutMs: number
  requestId: string
  accessToken?: string | null
  onUnauthorized?: (previousToken: string | null) => Promise<string | null> | string | null
  fetchImpl?: typeof fetch
  headers?: Record<string, string>
}

export interface UpstreamResult {
  status: number
  body: unknown
}

export function joinUpstreamUrl(baseUrl: string, path: string): string {
  const base = baseUrl.replace(/\/+$/, '')
  const suffix = path.startsWith('/') ? path : `/${path}`
  return base === '' ? suffix : `${base}${suffix}`
}

function isTimeoutError(error: unknown): boolean {
  if (typeof error !== 'object' || error === null) {
    return false
  }
  return (error as { name?: unknown }).name === 'TimeoutError'
}

function toTransportError(error: unknown, requestId: string, timeoutMs: number): AppError {
  if (isTimeoutError(error)) {
    return createAppError({
      code: 'upstream_timeout',
      message: `Upstream request timed out after ${timeoutMs}ms.`,
      requestId,
    })
  }
  return createAppError({
    code: 'upstream_unavailable',
    message: 'Upstream service is unavailable.',
    requestId,
  })
}

async function parseBody(response: Response): Promise<unknown> {
  if (response.status === 204) {
    return null
  }
  const text = await response.text()
  if (text.trim() === '') {
    return null
  }
  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}

export async function request(
  config: UpstreamRequestConfig,
  method: UpstreamMethod,
  path: string,
  payload?: unknown,
): Promise<UpstreamResult> {
  const fetchImpl = config.fetchImpl ?? fetch
  const url = joinUpstreamUrl(config.baseUrl, path)
  const initialToken = config.accessToken ?? null

  const doFetch = (token: string | null): Promise<Response> => {
    const headers: Record<string, string> = {
      accept: 'application/json',
      'x-request-id': config.requestId,
      ...config.headers,
    }
    if (token !== null && token !== '') {
      headers.authorization = `Bearer ${token}`
    }
    const init: RequestInit = {
      method,
      headers,
      signal: AbortSignal.timeout(config.timeoutMs),
    }
    if (payload !== undefined) {
      headers['content-type'] = 'application/json'
      init.body = JSON.stringify(payload)
    }
    return fetchImpl(url, init)
  }

  let response: Response
  try {
    response = await doFetch(initialToken)
    if (response.status === 401 && config.onUnauthorized !== undefined) {
      const refreshed = await config.onUnauthorized(initialToken)
      if (refreshed !== null && refreshed !== '' && refreshed !== initialToken) {
        response = await doFetch(refreshed)
      }
    }
  } catch (error) {
    throw toTransportError(error, config.requestId, config.timeoutMs)
  }

  const body = await parseBody(response)
  return { status: response.status, body }
}
