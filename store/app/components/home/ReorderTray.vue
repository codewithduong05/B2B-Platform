<script setup lang="ts">
import { ref } from 'vue'

interface ReorderItem {
  sku: string
  title: string
  lastOrder: string
  qty: number
}

const { addToCart } = useCart()
const items = ref<ReorderItem[]>([])

function incrementQty(item: ReorderItem) {
  item.qty++
}

function decrementQty(item: ReorderItem) {
  if (item.qty > 1) item.qty--
}

async function instantReorder(item: ReorderItem) {
  try {
    await addToCart(item.sku, item.qty)
  } catch {
    // Cart requires auth
  }
}
</script>

<template>
  <section class="reorder-section">
    <div class="reorder-header">
      <span class="material-symbols-outlined reorder-icon">history</span>
      <h2 class="section-title">Instant Replenishment</h2>
    </div>
    <div v-if="items.length === 0" class="reorder-empty">
      <span class="material-symbols-outlined reorder-empty-icon">receipt_long</span>
      <p class="reorder-empty-text">Your recent orders will appear here for quick reorder.</p>
      <NuxtLink to="/catalog" class="button button-secondary">Browse Catalog</NuxtLink>
    </div>
    <div v-else class="reorder-grid">
      <div v-for="item in items" :key="item.sku" class="reorder-card">
        <div class="reorder-info">
          <span class="reorder-sku">{{ item.sku }}</span>
          <span class="reorder-title">{{ item.title }}</span>
          <span class="reorder-last">Last ordered: {{ item.lastOrder }}</span>
        </div>
        <div class="reorder-controls">
          <div class="reorder-qty">
            <button class="qty-btn" @click="decrementQty(item)">-</button>
            <input v-model.number="item.qty" type="number" min="1" class="qty-input" />
            <button class="qty-btn" @click="incrementQty(item)">+</button>
          </div>
          <button class="button reorder-btn" @click="instantReorder(item)">
            <span class="material-symbols-outlined">autorenew</span>
            Reorder
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.reorder-section {
  margin-bottom: var(--space-xl);
}

.reorder-header {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  margin-bottom: var(--space-lg);
}

.reorder-icon {
  font-size: 22px;
  color: var(--secondary);
}

.section-title {
  font-size: var(--text-headline-md);
  font-weight: 700;
  color: var(--on-surface);
  letter-spacing: var(--tracking-headline-md);
  margin: 0;
}

.reorder-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-lg);
}

.reorder-card {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.reorder-info {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.reorder-sku {
  font-size: var(--text-tabular-sm);
  font-weight: 600;
  color: var(--secondary);
  font-variant-numeric: tabular-nums;
}

.reorder-title {
  font-size: var(--text-body-md);
  font-weight: 500;
  color: var(--on-surface);
}

.reorder-last {
  font-size: var(--text-body-sm);
  color: var(--muted);
}

.reorder-controls {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  margin-top: auto;
}

.reorder-qty {
  display: flex;
  align-items: center;
  background-color: var(--surface-container-low);
  border-radius: var(--radius);
}

.qty-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  font-size: var(--text-body-md);
  font-weight: 700;
  color: var(--muted);
  cursor: pointer;
  transition: color 0.15s ease;
}

.qty-btn:hover {
  color: var(--on-surface);
}

.qty-input {
  width: 48px;
  height: 32px;
  border: none;
  background: transparent;
  text-align: center;
  font-family: var(--font-family);
  font-size: var(--text-tabular-md);
  font-weight: 500;
  color: var(--on-surface);
  font-variant-numeric: tabular-nums;
  outline: none;
}

.reorder-btn {
  flex: 1;
  height: 36px;
}

.reorder-btn .material-symbols-outlined {
  font-size: 16px;
}

@media (max-width: 768px) {
  .reorder-grid {
    grid-template-columns: 1fr;
  }
}

.reorder-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-md);
  padding: var(--space-xl);
  background-color: var(--surface);
  border: 1px dashed var(--border);
  border-radius: var(--radius-lg);
  text-align: center;
}

.reorder-empty-icon {
  font-size: 40px;
  color: var(--outline-variant);
}

.reorder-empty-text {
  margin: 0;
  font-size: var(--text-body-sm);
  color: var(--muted);
}
</style>
