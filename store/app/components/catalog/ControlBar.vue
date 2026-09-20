<script setup lang="ts">
defineProps<{
  activeFilters: { key: string; label: string }[]
  sortBy: string
  viewMode: 'grid' | 'table'
}>()

const emit = defineEmits<{
  (e: 'remove-filter', key: string): void
  (e: 'sort', value: string): void
  (e: 'view-mode', mode: 'grid' | 'table'): void
}>()

const sortOptions = [
  { value: 'contract_low', label: 'Contract Price (Low to High)' },
  { value: 'stock_high', label: 'Stock Availability (Highest)' },
  { value: 'lead_time', label: 'Lead Time SLA (Fastest)' },
  { value: 'popular', label: 'Most Requisitioned' },
]
</script>

<template>
  <div class="control-bar">
    <div class="control-bar-left">
      <span class="control-bar-label">Active:</span>
      <span v-for="filter in activeFilters" :key="filter.key" class="control-pill">
        {{ filter.label }}
        <button class="control-pill-close" @click="emit('remove-filter', filter.key)">
          <span class="material-symbols-outlined">close</span>
        </button>
      </span>
      <span v-if="!activeFilters.length" class="control-bar-empty">No filters applied</span>
    </div>
    <div class="control-bar-right">
      <div class="control-sort">
        <span class="control-sort-label">Sort by:</span>
        <select
          :value="sortBy"
          class="control-sort-select"
          @change="emit('sort', ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="opt in sortOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </option>
        </select>
      </div>
      <div class="control-view-toggle">
        <button
          class="control-view-btn"
          :class="{ 'control-view-btn-active': viewMode === 'grid' }"
          title="Grid View"
          @click="emit('view-mode', 'grid')"
        >
          <span class="material-symbols-outlined">grid_view</span>
        </button>
        <button
          class="control-view-btn"
          :class="{ 'control-view-btn-active': viewMode === 'table' }"
          title="Table View"
          @click="emit('view-mode', 'table')"
        >
          <span class="material-symbols-outlined">table_rows</span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.control-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-md);
  padding: var(--space-md);
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  flex-wrap: wrap;
}

.control-bar-left {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  flex-wrap: wrap;
}

.control-bar-label {
  font-size: var(--text-label-sm);
  font-weight: 600;
  text-transform: uppercase;
  color: var(--muted);
  margin-right: var(--space-xs);
}

.control-bar-empty {
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.control-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: var(--radius);
  background-color: var(--surface-container-high);
  font-size: var(--text-label-sm);
  color: var(--on-surface);
}

.control-pill-close {
  border: none;
  background: transparent;
  padding: 0;
  cursor: pointer;
  display: flex;
  color: var(--muted);
}

.control-pill-close:hover {
  color: var(--error);
}

.control-pill-close .material-symbols-outlined {
  font-size: 14px;
}

.control-bar-right {
  display: flex;
  align-items: center;
  gap: var(--space-md);
}

.control-sort {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
}

.control-sort-label {
  font-size: var(--text-label-sm);
  color: var(--muted);
  white-space: nowrap;
}

.control-sort-select {
  appearance: none;
  background-color: var(--surface-container-low);
  border: none;
  border-radius: var(--radius);
  padding: 6px 28px 6px 12px;
  font-family: var(--font-family);
  font-size: var(--text-label-md);
  color: var(--on-surface);
  cursor: pointer;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' fill='%2345464d'%3E%3Cpath d='M6 8L1 3h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 8px center;
}

.control-view-toggle {
  display: flex;
  align-items: center;
  gap: 2px;
  background-color: var(--surface-container-low);
  padding: 2px;
  border-radius: var(--radius);
}

.control-view-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: var(--radius);
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
}

.control-view-btn:hover {
  color: var(--on-surface);
}

.control-view-btn-active {
  background-color: var(--surface);
  color: var(--secondary);
  box-shadow: var(--shadow-1);
}
</style>
