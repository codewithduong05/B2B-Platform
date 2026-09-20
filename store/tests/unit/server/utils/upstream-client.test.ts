import { describe, it, expect, vi, beforeEach } from 'vitest'
import { request, joinUpstreamUrl } from '#server/utils/upstream-client'
import type { UpstreamRequestConfig } from '#server/utils/upstream-client'

function createMockFetch(
  responses: Array<{ status: number; body?: unknown; headers?: Record<string, string> }>,
) {
  let callCount = 0
  return vi.fn(async (_input: unknown, _init?: RequestInit) => {
    const response = responses[Math.min(callCount, responses.length - 1)]
    callCount++
    if (response.status === 204 || response.body === undefined) {
      return new Response(null, {
        status: response.status,
        headers: response.headers,
      })
    }
    return new Response(JSON.stringify(response.body), {
      status: response.status,
      headers: { 'content-type': 'application/json', ...response.headers },
    })
  })
}

function createTimeoutFetch() {
  return vi.fn(async (_input: unknown, init?: RequestInit) => {
    return new Promise((_, reject) => {
      const signal = init?.signal
      if (signal) {
        signal.addEventListener('abort', () => {
          reject(new DOMException('The operation was aborted due to timeout', 'TimeoutError'))
        })
      }
    })
  })
}

describe('joinUpstreamUrl', () => {
  it('joins base and path correctly', () => {
    expect(joinUpstreamUrl('http://localhost:8000', '/api/v1/items')).toBe(
      'http://localhost:8000/api/v1/items',
    )
  })

  it('strips trailing slashes from base', () => {
    expect(joinUpstreamUrl('http://localhost:8000/', '/api/v1/items')).toBe(
      'http://localhost:8000/api/v1/items',
    )
  })

  it('adds leading slash to path if missing', () => {
    expect(joinUpstreamUrl('http://localhost:8000', 'api/v1/items')).toBe(
      'http://localhost:8000/api/v1/items',
    )
  })

  it('handles empty base', () => {
    expect(joinUpstreamUrl('', '/api/v1/items')).toBe('/api/v1/items')
  })
})

describe('request', () => {
  const baseConfig: UpstreamRequestConfig = {
    baseUrl: 'http://localhost:8000',
    timeoutMs: 5000,
    requestId: 'req_test_1',
  }

  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('makes a GET request and returns status/body', async () => {
    const fetchImpl = createMockFetch([{ status: 200, body: { id: 1, name: 'Item' } }])
    const result = await request({ ...baseConfig, fetchImpl }, 'GET', '/api/v1/items/1')
    expect(result).toEqual({ status: 200, body: { id: 1, name: 'Item' } })
    expect(fetchImpl).toHaveBeenCalledOnce()
  })

  it('sends authorization header when accessToken is provided', async () => {
    const fetchImpl = createMockFetch([{ status: 200, body: {} }])
    await request({ ...baseConfig, accessToken: 'tok_abc', fetchImpl }, 'GET', '/api/v1/me')
    const [, init] = fetchImpl.mock.calls[0]
    expect(init?.headers).toMatchObject({ authorization: 'Bearer tok_abc' })
  })

  it('does not send authorization header when accessToken is null', async () => {
    const fetchImpl = createMockFetch([{ status: 200, body: {} }])
    await request({ ...baseConfig, accessToken: null, fetchImpl }, 'GET', '/api/v1/me')
    const [, init] = fetchImpl.mock.calls[0]
    expect((init?.headers as Record<string, string>)?.authorization).toBeUndefined()
  })

  it('sends content-type and body for POST with payload', async () => {
    const fetchImpl = createMockFetch([{ status: 201, body: { id: 2 } }])
    await request({ ...baseConfig, fetchImpl }, 'POST', '/api/v1/items', { name: 'New' })
    const [, init] = fetchImpl.mock.calls[0]
    expect(init?.headers).toMatchObject({ 'content-type': 'application/json' })
    expect(init?.body).toBe(JSON.stringify({ name: 'New' }))
  })

  it('does not send content-type for GET without payload', async () => {
    const fetchImpl = createMockFetch([{ status: 200, body: {} }])
    await request({ ...baseConfig, fetchImpl }, 'GET', '/api/v1/items')
    const [, init] = fetchImpl.mock.calls[0]
    expect((init?.headers as Record<string, string>)?.['content-type']).toBeUndefined()
  })

  it('returns null body for 204', async () => {
    const fetchImpl = createMockFetch([{ status: 204 }])
    const result = await request({ ...baseConfig, fetchImpl }, 'DELETE', '/api/v1/items/1')
    expect(result).toEqual({ status: 204, body: null })
  })

  it('returns null body for empty response', async () => {
    const fetchImpl = createMockFetch([{ status: 200, body: undefined }])
    const result = await request({ ...baseConfig, fetchImpl }, 'GET', '/api/v1/empty')
    expect(result).toEqual({ status: 200, body: null })
  })

  it('retries on 401 when onUnauthorized returns new token', async () => {
    const fetchImpl = createMockFetch([
      { status: 401, body: { detail: 'Unauthorized', code: 'unauthorized', request_id: 'up_1' } },
      { status: 200, body: { ok: true } },
    ])
    const onUnauthorized = vi.fn().mockResolvedValue('tok_refreshed')
    const result = await request(
      { ...baseConfig, accessToken: 'tok_old', onUnauthorized, fetchImpl },
      'GET',
      '/api/v1/me',
    )
    expect(onUnauthorized).toHaveBeenCalledWith('tok_old')
    expect(fetchImpl).toHaveBeenCalledTimes(2)
    const [, secondInit] = fetchImpl.mock.calls[1]
    expect(secondInit?.headers).toMatchObject({ authorization: 'Bearer tok_refreshed' })
    expect(result).toEqual({ status: 200, body: { ok: true } })
  })

  it('does not retry when onUnauthorized returns null', async () => {
    const fetchImpl = createMockFetch([{ status: 401, body: {} }])
    const onUnauthorized = vi.fn().mockResolvedValue(null)
    const result = await request(
      { ...baseConfig, accessToken: 'tok_old', onUnauthorized, fetchImpl },
      'GET',
      '/api/v1/me',
    )
    expect(fetchImpl).toHaveBeenCalledOnce()
    expect(result.status).toBe(401)
  })

  it('does not retry when refreshed token equals initial token', async () => {
    const fetchImpl = createMockFetch([{ status: 401, body: {} }])
    const onUnauthorized = vi.fn().mockResolvedValue('tok_old')
    await request(
      { ...baseConfig, accessToken: 'tok_old', onUnauthorized, fetchImpl },
      'GET',
      '/api/v1/me',
    )
    expect(fetchImpl).toHaveBeenCalledOnce()
  })

  it('throws upstream_timeout on TimeoutError', async () => {
    const fetchImpl = createTimeoutFetch()
    await expect(
      request({ ...baseConfig, fetchImpl, timeoutMs: 100 }, 'GET', '/api/v1/slow'),
    ).rejects.toMatchObject({
      code: 'upstream_timeout',
      message: 'Upstream request timed out after 100ms.',
      requestId: 'req_test_1',
    })
  })

  it('throws upstream_unavailable on other fetch errors', async () => {
    const fetchImpl = vi.fn().mockRejectedValue(new Error('Network error'))
    await expect(
      request({ ...baseConfig, fetchImpl }, 'GET', '/api/v1/down'),
    ).rejects.toMatchObject({
      code: 'upstream_unavailable',
      message: 'Upstream service is unavailable.',
      requestId: 'req_test_1',
    })
  })
})
