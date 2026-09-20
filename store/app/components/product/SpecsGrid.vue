<script setup lang="ts">
import type { SpecRow } from '~~/shared/product'

defineProps<{ specs: SpecRow[]; standard?: string }>()
</script>

<template>
  <div class="specs-grid">
    <div class="specs-header">
      <div class="specs-header-left">
        <span class="material-symbols-outlined specs-header-icon">tune</span>
        <span class="specs-header-title">Mechanical &amp; Electrical Parameters</span>
      </div>
      <span v-if="standard" class="specs-header-standard">{{ standard }}</span>
    </div>
    <div class="specs-table">
      <div
        v-for="(spec, i) in specs"
        :key="i"
        class="specs-row"
        :class="{ 'specs-row-even': i % 2 === 0 }"
      >
        <span class="specs-label">{{ spec.label }}</span>
        <span class="specs-value" :class="{ 'specs-value-highlight': spec.highlight }">
          {{ spec.value }}
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.specs-grid {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: var(--space-lg);
}

.specs-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: var(--space-sm);
  margin-bottom: var(--space-md);
}

.specs-header-left {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.specs-header-icon {
  font-size: 20px;
  color: var(--secondary);
}

.specs-header-title {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
}

.specs-header-standard {
  font-size: var(--text-label-sm);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: var(--tracking-label-sm);
  color: var(--muted);
}

.specs-table {
  display: flex;
  flex-direction: column;
  gap: 2px;
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.specs-row {
  display: grid;
  grid-template-columns: 1fr 1.5fr;
  padding: 10px var(--space-md);
  align-items: center;
  background-color: var(--surface-container-low);
}

.specs-row-even {
  background-color: var(--surface);
}

.specs-label {
  font-size: var(--text-body-sm);
  font-weight: 500;
  color: var(--muted);
}

.specs-value {
  font-size: var(--text-body-md);
  color: var(--on-surface);
}

.specs-value-highlight {
  font-size: var(--text-tabular-md);
  font-weight: 600;
  color: var(--secondary);
}
</style>
