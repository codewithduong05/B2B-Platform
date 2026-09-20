<script setup lang="ts">
defineProps<{
  supplier: {
    name: string
    id: string
    tier: string
    audited: boolean
    onTimeSla: string
    qualityDefect: string
    terms: string
    compliance: string[]
  }
}>()
</script>

<template>
  <div class="supplier-card">
    <!-- Header -->
    <div class="supplier-header">
      <div class="supplier-header-left">
        <div class="supplier-icon-box">
          <span class="material-symbols-outlined">factory</span>
        </div>
        <div class="supplier-info">
          <span class="supplier-name">{{ supplier.name }}</span>
          <span class="supplier-id">Vendor ID: {{ supplier.id }} • {{ supplier.tier }}</span>
        </div>
      </div>
      <span v-if="supplier.audited" class="supplier-audited">Audited</span>
    </div>

    <!-- Metrics -->
    <div class="supplier-metrics">
      <div class="supplier-metric">
        <span class="supplier-metric-label">On-Time SLA</span>
        <span class="supplier-metric-value">{{ supplier.onTimeSla }}</span>
      </div>
      <div class="supplier-metric">
        <span class="supplier-metric-label">Quality Defect</span>
        <span class="supplier-metric-value">{{ supplier.qualityDefect }}</span>
      </div>
      <div class="supplier-metric">
        <span class="supplier-metric-label">Terms</span>
        <span class="supplier-metric-value supplier-metric-value-accent">{{ supplier.terms }}</span>
      </div>
    </div>

    <!-- Compliance -->
    <div class="supplier-compliance">
      <div v-for="(item, i) in supplier.compliance" :key="i" class="supplier-compliance-item">
        <span class="material-symbols-outlined supplier-compliance-icon">verified</span>
        <span class="supplier-compliance-text">{{ item }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.supplier-card {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.supplier-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.supplier-header-left {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.supplier-icon-box {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-lg);
  background-color: var(--surface-container);
  display: flex;
  align-items: center;
  justify-content: center;
}

.supplier-icon-box .material-symbols-outlined {
  font-size: 22px;
  color: var(--secondary);
}

.supplier-info {
  display: flex;
  flex-direction: column;
}

.supplier-name {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
  line-height: 1.3;
}

.supplier-id {
  font-size: var(--text-label-sm);
  color: var(--muted);
  font-family: monospace;
}

.supplier-audited {
  padding: 2px 8px;
  border-radius: var(--radius);
  background-color: var(--surface-container-high);
  font-size: var(--text-label-sm);
  font-weight: 700;
  color: var(--secondary);
  text-transform: uppercase;
}

/* ── Metrics ── */
.supplier-metrics {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-xs);
}

.supplier-metric {
  padding: var(--space-sm);
  border-radius: var(--radius);
  background-color: var(--surface-container-low);
  display: flex;
  flex-direction: column;
}

.supplier-metric-label {
  font-size: 10px;
  font-weight: 500;
  text-transform: uppercase;
  color: var(--muted);
}

.supplier-metric-value {
  font-size: var(--text-tabular-md);
  font-weight: 700;
  color: var(--on-surface);
}

.supplier-metric-value-accent {
  color: var(--secondary);
}

/* ── Compliance ── */
.supplier-compliance {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.supplier-compliance-item {
  display: flex;
  align-items: flex-start;
  gap: var(--space-sm);
  font-size: var(--text-body-sm);
  color: var(--muted);
}

.supplier-compliance-icon {
  font-size: 16px;
  color: var(--secondary);
  flex-shrink: 0;
  margin-top: 2px;
}
</style>
