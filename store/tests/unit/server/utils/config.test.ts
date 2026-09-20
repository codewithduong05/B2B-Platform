import { describe, it, expect, vi } from 'vitest'

vi.hoisted(() => {
  vi.stubGlobal('useRuntimeConfig', () => ({
    api: {
      baseUrl: 'http://localhost:8000',
      timeoutMs: 5000,
    },
  }))
})

import { resolveUpstreamConfig } from '#server/utils/config'

describe('resolveUpstreamConfig', () => {
  it('returns normalized config from useRuntimeConfig', () => {
    const config = resolveUpstreamConfig()
    expect(config).toHaveProperty('baseUrl')
    expect(config).toHaveProperty('timeoutMs')
    expect(typeof config.baseUrl).toBe('string')
    expect(typeof config.timeoutMs).toBe('number')
    expect(config.timeoutMs).toBeGreaterThan(0)
  })
})
