import { test, expect } from '@playwright/test'

test.describe('Foundation E2E', () => {
  test('GET /api/health echoes x-request-id', async ({ request }) => {
    const response = await request.get('/api/health', {
      headers: { 'x-request-id': 'e2e-health-001', accept: 'application/json' },
    })
    expect(response.ok()).toBeTruthy()
    const body = await response.json()
    expect(body.status).toBe('ok')
    expect(body.requestId).toBe('e2e-health-001')
    expect(response.headers()['x-request-id']).toBe('e2e-health-001')
  })

  test('GET /api/health generates x-request-id when not provided', async ({ request }) => {
    const response = await request.get('/api/health', {
      headers: { accept: 'application/json' },
    })
    expect(response.ok()).toBeTruthy()
    const body = await response.json()
    expect(body.status).toBe('ok')
    expect(body.requestId).toMatch(/^req_/)
    expect(response.headers()['x-request-id']).toBe(body.requestId)
  })

  test('GET /api/session returns 401 without cookie', async ({ request }) => {
    const response = await request.get('/api/session', {
      headers: { 'x-request-id': 'e2e-session-001', accept: 'application/json' },
    })
    expect(response.status()).toBe(401)
    const body = await response.json()
    expect(body.statusCode).toBe(401)
    expect(body.data).toEqual({
      code: 'unauthorized',
      detail: 'Active session is required.',
      request_id: 'e2e-session-001',
    })
  })
})
