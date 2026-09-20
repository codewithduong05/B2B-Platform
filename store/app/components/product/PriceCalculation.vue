<script setup lang="ts">
import type { PriceTier } from '~~/shared/product'

defineProps<{
  unitPrice: number
  tiers: PriceTier[]
  unit: string
  activeTierIndex: number
  qty: number
  subtotal: number
}>()

const emit = defineEmits<{
  (e: 'update:qty', val: number): void
  (e: 'adjust', delta: number): void
  (e: 'set', val: number): void
}>()
</script>

<template>
  <div class="price-calc">
    <!-- Live Price Ribbon -->
    <div class="price-ribbon">
      <div class="price-ribbon-left">
        <span class="price-ribbon-label">Enterprise Net Price</span>
        <div class="price-ribbon-row">
          <span class="price-ribbon-value">${{ unitPrice.toFixed(2) }}</span>
          <span class="price-ribbon-unit">/ {{ unit }}</span>
        </div>
      </div>
      <div class="price-ribbon-right">
        <span class="price-ribbon-label">Subtotal (Pre-Tax)</span>
        <span class="price-ribbon-subtotal">${{ subtotal.toFixed(2) }}</span>
      </div>
    </div>

    <!-- Volume Tiers -->
    <div class="price-tiers">
      <div class="price-tiers-header">
        <span class="price-tiers-title">Volume Tiered Discount Schedule</span>
        <span class="price-tiers-locked">Contract Tier 1 Locked</span>
      </div>
      <div class="price-tiers-table">
        <div
          v-for="(tier, i) in tiers"
          :key="tier.id"
          class="price-tier-row"
          :class="{ 'price-tier-active': i === activeTierIndex }"
        >
          <span class="price-tier-range">
            <span
              class="price-tier-dot"
              :class="{ 'price-tier-dot-active': i === activeTierIndex }"
            ></span>
            {{ tier.range }} units
          </span>
          <span class="price-tier-price">${{ tier.price.toFixed(2) }}</span>
          <span class="price-tier-discount">{{ tier.discount }}</span>
        </div>
      </div>
    </div>

    <!-- Quantity Picker -->
    <div class="price-qty">
      <div class="price-qty-header">
        <span class="price-qty-label">Order Quantity</span>
        <span class="price-qty-hint">MOQ: 1 unit • Increments of 1</span>
      </div>
      <div class="price-qty-controls">
        <div class="price-qty-stepper">
          <button class="qty-btn" @click="emit('adjust', -1)">
            <span class="material-symbols-outlined">remove</span>
          </button>
          <input
            :value="qty"
            type="number"
            min="1"
            class="qty-input"
            @input="emit('update:qty', Number(($event.target as HTMLInputElement).value))"
          />
          <button class="qty-btn" @click="emit('adjust', 1)">
            <span class="material-symbols-outlined">add</span>
          </button>
        </div>
        <div class="price-qty-presets">
          <button class="preset-btn" @click="emit('set', 10)">+10</button>
          <button class="preset-btn" @click="emit('set', 50)">+50</button>
          <button class="preset-btn" @click="emit('set', 100)">+100</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.price-calc {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: var(--space-lg);
}

/* ── Price Ribbon ── */
.price-ribbon {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  padding: var(--space-md);
  background-color: var(--surface-container-low);
  border-radius: var(--radius-lg);
}

.price-ribbon-left,
.price-ribbon-right {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.price-ribbon-label {
  font-size: var(--text-label-sm);
  font-weight: 600;
  text-transform: uppercase;
  color: var(--muted);
}

.price-ribbon-row {
  display: flex;
  align-items: baseline;
  gap: var(--space-xs);
}

.price-ribbon-value {
  font-size: 2rem;
  font-weight: 700;
  color: var(--on-surface);
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.price-ribbon-unit {
  font-size: var(--text-body-sm);
  color: var(--muted);
}

.price-ribbon-subtotal {
  font-size: var(--text-tabular-lg);
  font-weight: 700;
  color: var(--on-surface);
  font-variant-numeric: tabular-nums;
}

/* ── Tiers ── */
.price-tiers {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.price-tiers-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.price-tiers-title {
  font-size: var(--text-label-sm);
  font-weight: 600;
  text-transform: uppercase;
  color: var(--on-surface);
}

.price-tiers-locked {
  font-size: var(--text-label-sm);
  color: var(--secondary);
}

.price-tiers-table {
  border-radius: var(--radius-lg);
  overflow: hidden;
  background-color: var(--surface-container);
}

.price-tier-row {
  display: grid;
  grid-template-columns: 5fr 4fr 3fr;
  padding: var(--space-sm) var(--space-md);
  font-size: var(--text-body-sm);
  font-weight: 500;
  align-items: center;
  transition: background-color 0.15s ease;
}

.price-tier-row:nth-child(odd) {
  background-color: var(--surface);
}

.price-tier-row:nth-child(even) {
  background-color: var(--surface-container-low);
}

.price-tier-active {
  background-color: var(--secondary-fixed) !important;
  color: var(--on-secondary-fixed);
}

.price-tier-range {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.price-tier-dot {
  width: 6px;
  height: 6px;
  border-radius: var(--radius-full);
  background-color: transparent;
}

.price-tier-dot-active {
  background-color: var(--secondary);
}

.price-tier-price {
  text-align: right;
  font-size: var(--text-tabular-sm);
  font-weight: 600;
}

.price-tier-discount {
  text-align: right;
  font-size: var(--text-label-sm);
}

/* ── Quantity ── */
.price-qty {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.price-qty-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: var(--text-label-sm);
}

.price-qty-label {
  font-weight: 600;
  color: var(--on-surface);
}

.price-qty-hint {
  color: var(--muted);
}

.price-qty-controls {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.price-qty-stepper {
  display: flex;
  align-items: center;
  background-color: var(--surface-container-low);
  border-radius: var(--radius-lg);
  padding: 4px;
}

.qty-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: var(--radius);
  background: transparent;
  color: var(--muted);
  cursor: pointer;
}

.qty-btn:hover {
  color: var(--on-surface);
  background-color: var(--surface-container);
}

.qty-input {
  width: 80px;
  height: 36px;
  border: none;
  background: transparent;
  text-align: center;
  font-family: var(--font-family);
  font-size: var(--text-tabular-lg);
  font-weight: 700;
  color: var(--on-surface);
  font-variant-numeric: tabular-nums;
  outline: none;
}

.price-qty-presets {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 4px;
  flex: 1;
}

.preset-btn {
  padding: var(--space-sm);
  border: none;
  border-radius: var(--radius);
  background-color: var(--surface-container-low);
  font-family: var(--font-family);
  font-size: var(--text-label-md);
  font-weight: 500;
  color: var(--on-surface);
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.preset-btn:hover {
  background-color: var(--surface-container);
}
</style>
