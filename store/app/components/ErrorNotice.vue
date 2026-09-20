<script setup lang="ts">
import { computed } from 'vue'
import type { AppError } from '../../shared/errors'

const props = defineProps<{ error: AppError }>()

const fields = computed<[string, string][]>(() => {
  if (!props.error.fields) {
    return []
  }
  return Object.entries(props.error.fields)
})
</script>

<template>
  <section class="notice notice-error" role="alert" aria-live="assertive">
    <h2>Something went wrong</h2>
    <p class="muted">{{ error.code }} · {{ error.requestId }}</p>
    <p>{{ error.message }}</p>
    <ul v-if="fields.length" class="notice-fields">
      <li v-for="[name, value] in fields" :key="name" class="notice-field">
        <span class="notice-field-key">{{ name }}</span>
        <span class="notice-field-value">{{ value }}</span>
      </li>
    </ul>
  </section>
</template>
