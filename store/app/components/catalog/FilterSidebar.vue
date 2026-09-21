<script setup lang="ts">
import type { CatalogFilters } from '~~/shared/catalog'

const props = defineProps<{
  filters: CatalogFilters
}>()

const emit = defineEmits<{
  (e: 'update:filters', value: Partial<CatalogFilters>): void
  (e: 'reset'): void
}>()

const suppliers = ref<Array<{ name: string; count: number; checked: boolean }>>([])

const leadTimes: Array<{ value: string; label: string; count: number }> = []

const warehouses: Array<{ name: string; checked: boolean }> = []

const complianceTags: Array<{ name: string; active: boolean }> = []

// Fetch suppliers from BFF
async function loadSuppliers() {
  try {
    const data = await $fetch<{ suppliers: Array<{ name: string; code: string }> }>('/api/catalog/filters')
    suppliers.value = data.suppliers.map((s) => ({
      name: s.name,
      count: 0,
      checked: false,
    }))
  } catch {
    suppliers.value = []
  }
}

onMounted(() => {
  loadSuppliers()
})

const moqDisplay = ref(props.filters.moqMax)

function onMoqChange(e: Event) {
  const val = Number((e.target as HTMLInputElement).value)
  moqDisplay.value = val
  emit('update:filters', { moqMax: val })
}

function toggleSupplier(name: string) {
  const current = [...props.filters.suppliers]
  const idx = current.indexOf(name)
  if (idx >= 0) current.splice(idx, 1)
  else current.push(name)
  emit('update:filters', { suppliers: current })
}

function setLeadTime(value: string) {
  emit('update:filters', { leadTime: value })
}

function toggleWarehouse(name: string) {
  const current = [...props.filters.warehouses]
  const idx = current.indexOf(name)
  if (idx >= 0) current.splice(idx, 1)
  else current.push(name)
  emit('update:filters', { warehouses: current })
}

function toggleCompliance(name: string) {
  const current = [...props.filters.compliance]
  const idx = current.indexOf(name)
  if (idx >= 0) current.splice(idx, 1)
  else current.push(name)
  emit('update:filters', { compliance: current })
}
</script>

<template>
  <aside class="filter-sidebar">
    <div class="filter-header">
      <span class="material-symbols-outlined filter-header-icon">tune</span>
      <span class="filter-header-title">Filter Products</span>
      <button class="filter-reset" @click="emit('reset')">Reset All</button>
    </div>

    <!-- Supplier Facet -->
    <div class="filter-facet">
      <span class="filter-facet-label">Enterprise Supplier</span>
      <div class="filter-checkbox-group">
        <label v-for="s in suppliers" :key="s.name" class="filter-checkbox-label">
          <input
            type="checkbox"
            class="filter-checkbox"
            :checked="filters.suppliers.includes(s.name)"
            @change="toggleSupplier(s.name)"
          />
          <span class="filter-checkbox-text">{{ s.name }}</span>
          <span class="filter-checkbox-count">{{ s.count }}</span>
        </label>
      </div>
    </div>

    <!-- Lead Time Facet -->
    <div class="filter-facet">
      <span class="filter-facet-label">Lead Time SLA</span>
      <div class="filter-radio-group">
        <label v-for="lt in leadTimes" :key="lt.value" class="filter-radio-label">
          <input
            type="radio"
            name="leadtime"
            class="filter-radio"
            :checked="filters.leadTime === lt.value"
            @change="setLeadTime(lt.value)"
          />
          <span class="filter-radio-text">{{ lt.label }}</span>
          <span class="filter-radio-count">{{ lt.count }}</span>
        </label>
      </div>
    </div>

    <!-- Warehouse Facet -->
    <div class="filter-facet">
      <span class="filter-facet-label">Warehouse Allocation</span>
      <div class="filter-checkbox-group">
        <label v-for="w in warehouses" :key="w.name" class="filter-checkbox-label">
          <input
            type="checkbox"
            class="filter-checkbox"
            :checked="filters.warehouses.includes(w.name)"
            @change="toggleWarehouse(w.name)"
          />
          <span class="filter-checkbox-text">{{ w.name }}</span>
        </label>
      </div>
    </div>

    <!-- MOQ Slider -->
    <div class="filter-facet">
      <div class="filter-facet-header">
        <span class="filter-facet-label">Max MOQ Threshold</span>
        <span class="filter-moq-value">{{ moqDisplay }} units</span>
      </div>
      <input
        type="range"
        :value="moqDisplay"
        min="1"
        max="100"
        class="filter-slider"
        @input="onMoqChange"
      />
      <div class="filter-slider-labels">
        <span>1 unit</span>
        <span>100 units</span>
      </div>
    </div>

    <!-- Compliance Tags -->
    <div class="filter-facet">
      <span class="filter-facet-label">Regulatory Compliance</span>
      <div class="filter-tags">
        <button
          v-for="tag in complianceTags"
          :key="tag.name"
          class="filter-tag"
          :class="{ 'filter-tag-active': filters.compliance.includes(tag.name) }"
          @click="toggleCompliance(tag.name)"
        >
          {{ tag.name }}
        </button>
      </div>
    </div>

    <!-- Contract Guarantee -->
    <div class="filter-contract">
      <span class="material-symbols-outlined filter-contract-icon">verified_user</span>
      <div class="filter-contract-text">
        <span class="filter-contract-title">Contract Price Lock</span>
        <span class="filter-contract-desc">
          All shown rates honor Master Agreement pricing index through Q4 2025.
        </span>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.filter-sidebar {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-md);
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

.filter-header {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding-bottom: var(--space-xs);
  background-color: var(--surface-container-low);
  margin: calc(-1 * var(--space-md)) calc(-1 * var(--space-md)) 0;
  padding: var(--space-sm) var(--space-md);
  border-radius: var(--radius-lg) var(--radius-lg) 0 0;
}

.filter-header-icon {
  font-size: 18px;
  color: var(--secondary);
}

.filter-header-title {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
}

.filter-reset {
  margin-left: auto;
  border: none;
  background: transparent;
  font-family: var(--font-family);
  font-size: var(--text-label-sm);
  font-weight: 600;
  color: var(--secondary);
  text-transform: uppercase;
  cursor: pointer;
}

.filter-reset:hover {
  color: var(--on-surface);
}

/* ── Facets ── */
.filter-facet {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.filter-facet-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.filter-facet-label {
  font-size: var(--text-label-md);
  font-weight: 600;
  color: var(--on-surface);
}

.filter-moq-value {
  font-size: var(--text-tabular-md);
  font-weight: 700;
  color: var(--secondary);
  font-variant-numeric: tabular-nums;
}

/* ── Checkboxes ── */
.filter-checkbox-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 160px;
  overflow-y: auto;
}

.filter-checkbox-label {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-xs);
  border-radius: var(--radius);
  cursor: pointer;
  font-size: var(--text-body-md);
  color: var(--on-surface);
  transition: background-color 0.1s ease;
}

.filter-checkbox-label:hover {
  background-color: var(--surface-container-low);
}

.filter-checkbox {
  width: 16px;
  height: 16px;
  accent-color: var(--secondary);
  flex-shrink: 0;
}

.filter-checkbox-text {
  flex: 1;
}

.filter-checkbox-count {
  font-size: var(--text-tabular-sm);
  font-weight: 500;
  color: var(--muted);
  font-variant-numeric: tabular-nums;
}

/* ── Radios ── */
.filter-radio-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.filter-radio-label {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-xs);
  border-radius: var(--radius);
  cursor: pointer;
  font-size: var(--text-body-md);
  color: var(--on-surface);
  transition: background-color 0.1s ease;
}

.filter-radio-label:hover {
  background-color: var(--surface-container-low);
}

.filter-radio {
  width: 16px;
  height: 16px;
  accent-color: var(--secondary);
  flex-shrink: 0;
}

.filter-radio-text {
  flex: 1;
}

.filter-radio-count {
  font-size: var(--text-tabular-sm);
  font-weight: 500;
  color: var(--muted);
  padding: 2px 6px;
  border-radius: var(--radius);
  background-color: var(--surface-container);
  font-variant-numeric: tabular-nums;
}

/* ── Slider ── */
.filter-slider {
  width: 100%;
  height: 6px;
  accent-color: var(--secondary);
  cursor: pointer;
}

.filter-slider-labels {
  display: flex;
  justify-content: space-between;
  font-size: var(--text-tabular-sm);
  color: var(--muted);
}

/* ── Tags ── */
.filter-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.filter-tag {
  padding: 4px 10px;
  border-radius: var(--radius);
  border: 1px solid var(--border);
  background-color: var(--surface-container);
  font-family: var(--font-family);
  font-size: var(--text-label-sm);
  color: var(--muted);
  cursor: pointer;
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
}

.filter-tag:hover {
  background-color: var(--surface-container-high);
}

.filter-tag-active {
  background-color: var(--surface-container-high);
  color: var(--secondary);
  font-weight: 600;
}

.filter-tag-active:hover {
  background-color: var(--secondary);
  color: var(--on-secondary);
}

/* ── Contract ── */
.filter-contract {
  display: flex;
  align-items: flex-start;
  gap: var(--space-sm);
  padding: var(--space-sm);
  background-color: var(--surface-container-low);
  border-radius: var(--radius-lg);
  margin-top: var(--space-xs);
}

.filter-contract-icon {
  font-size: 20px;
  color: var(--secondary);
  flex-shrink: 0;
}

.filter-contract-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.filter-contract-title {
  font-size: var(--text-label-md);
  font-weight: 700;
  color: var(--on-surface);
}

.filter-contract-desc {
  font-size: var(--text-body-sm);
  color: var(--muted);
}
</style>
