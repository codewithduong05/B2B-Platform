export interface UpstreamApiConfig {
  baseUrl: string
  timeoutMs: number
}

const DEFAULT_TIMEOUT_MS = 10_000

function normalizeTimeout(value: unknown): number {
  if (typeof value === 'number') {
    return Number.isFinite(value) && value > 0 ? value : DEFAULT_TIMEOUT_MS
  }
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    return Number.isFinite(parsed) && parsed > 0 ? parsed : DEFAULT_TIMEOUT_MS
  }
  return DEFAULT_TIMEOUT_MS
}

export function resolveUpstreamConfig(): UpstreamApiConfig {
  const runtime = useRuntimeConfig() as {
    api?: { baseUrl?: unknown; timeoutMs?: unknown }
  }
  const rawBaseUrl = runtime.api?.baseUrl
  const baseUrl = typeof rawBaseUrl === 'string' ? rawBaseUrl.replace(/\/+$/, '') : ''
  return { baseUrl, timeoutMs: normalizeTimeout(runtime.api?.timeoutMs) }
}
