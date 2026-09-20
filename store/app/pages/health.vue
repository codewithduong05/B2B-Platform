<script setup lang="ts">
import { computed } from 'vue'
import ErrorNotice from '~/components/ErrorNotice.vue'
import { createAppError, isWireError, statusCodeToErrorCode } from '../../shared/errors'
import type { AppError } from '../../shared/errors'

definePageMeta({ title: 'Health' })

interface HealthPayload {
  status: 'ok'
  requestId: string
}

const { data, error, pending } = await useFetch<HealthPayload>('/api/health')

const appError = computed<AppError | null>(() => {
  if (!error.value) {
    return null
  }
  const err = error.value
  const wire = (err as { data?: unknown }).data
  if (isWireError(wire)) {
    return createAppError({
      code: wire.code,
      message: wire.detail,
      requestId: wire.request_id,
    })
  }
  const statusCode = (err as { statusCode?: number }).statusCode ?? 500
  return createAppError({
    code: statusCodeToErrorCode(statusCode),
    message: 'Health check failed',
    requestId: String((err as { statusMessage?: string }).statusMessage ?? statusCode),
  })
})
</script>

<template>
  <div class="panel">
    <h1 class="panel-header">Service status</h1>
    <p v-if="pending" class="muted">Checking service health…</p>
    <template v-else-if="data">
      <p class="status-ok">Service is operational.</p>
      <p class="muted">Request: {{ data.requestId }}</p>
    </template>
    <ErrorNotice v-else-if="appError" :error="appError" />
    <p v-else class="muted">Health status is unavailable.</p>
  </div>
</template>
