<script setup lang="ts">
import type { StockLocation } from '~~/shared/product'

defineProps<{
  locations: StockLocation[]
  totalStock: number
}>()
</script>

<template>
  <div class="stock-radar">
    <div class="stock-radar-header">
      <div class="stock-radar-title">
        <span class="material-symbols-outlined">inventory_2</span>
        <span>Multi-Warehouse Fulfillable Stock</span>
      </div>
      <span class="stock-radar-total">{{ totalStock.toLocaleString() }} Units Global</span>
    </div>
    <div class="stock-radar-list">
      <div v-for="loc in locations" :key="loc.warehouse" class="stock-location">
        <div class="stock-location-left">
          <span class="stock-dot" :class="{ 'stock-dot-priority': loc.priority }"></span>
          <div class="stock-location-info">
            <span class="stock-location-name">{{ loc.warehouse }}</span>
            <span
              class="stock-location-detail"
              :class="{ 'stock-location-detail-priority': loc.priority }"
            >
              {{ loc.detail }}
            </span>
          </div>
        </div>
        <div class="stock-location-right">
          <span class="stock-location-units">{{ loc.units }} units</span>
          <span class="stock-location-transit">{{ loc.transit }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.stock-radar {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.stock-radar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.stock-radar-title {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-label-sm);
  font-weight: 600;
  text-transform: uppercase;
}

.stock-radar-title .material-symbols-outlined {
  font-size: 15px;
  color: var(--secondary);
}

.stock-radar-total {
  font-size: var(--text-tabular-sm);
  font-weight: 700;
  color: var(--on-surface);
  font-variant-numeric: tabular-nums;
}

.stock-radar-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.stock-location {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-sm);
  border-radius: var(--radius-lg);
  background-color: var(--surface-container-low);
}

.stock-location-left {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.stock-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  background-color: var(--secondary);
  flex-shrink: 0;
}

.stock-dot-priority {
  background-color: var(--secondary);
}

.stock-location-info {
  display: flex;
  flex-direction: column;
}

.stock-location-name {
  font-size: var(--text-label-md);
  font-weight: 600;
  color: var(--on-surface);
}

.stock-location-detail {
  font-size: var(--text-body-sm);
  color: var(--muted);
}

.stock-location-detail-priority {
  color: var(--secondary);
}

.stock-location-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.stock-location-units {
  font-size: var(--text-tabular-md);
  font-weight: 700;
  color: var(--on-surface);
  font-variant-numeric: tabular-nums;
}

.stock-location-transit {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--muted);
}
</style>
