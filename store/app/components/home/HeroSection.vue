<script setup lang="ts">
import { ref, onMounted } from 'vue'

const { addToCart } = useCart()

const stagedCount = ref(0)
const skuInput = ref('')
const qtyInput = ref(5)
const adding = ref(false)

const stats = ref({ products: 0, categories: 0, brands: 0, suppliers: 0 })

async function loadStats() {
  try {
    const data = await $fetch<{ products: number; categories: number; brands: number; suppliers: number }>('/api/home/stats')
    stats.value = data
  } catch {
    // Stats unavailable
  }
}

onMounted(() => {
  loadStats()
})

async function stageItem() {
  if (!skuInput.value.trim()) return
  adding.value = true
  try {
    await addToCart(skuInput.value.trim(), qtyInput.value)
    stagedCount.value++
    skuInput.value = ''
    qtyInput.value = 5
  } catch {
    // Cart unavailable (auth required)
  } finally {
    adding.value = false
  }
}

function presetSku(sku: string) {
  skuInput.value = sku
  qtyInput.value = 5
}
</script>

<template>
  <section class="hero-section">
    <div class="hero-status-bar">
      <span class="material-symbols-outlined hero-status-icon">shield_lock</span>
      <span class="hero-status-text">Session Secured</span>
    </div>

    <div class="hero-grid">
      <!-- KPIs -->
      <div class="hero-kpis">
        <div class="kpi-card">
          <span class="kpi-label">Products</span>
          <span class="kpi-value">{{ stats.products.toLocaleString() }}</span>
        </div>
        <div class="kpi-card">
          <span class="kpi-label">Categories</span>
          <span class="kpi-value">{{ stats.categories.toLocaleString() }}</span>
        </div>
        <div class="kpi-card">
          <span class="kpi-label">Brands</span>
          <span class="kpi-value">{{ stats.brands.toLocaleString() }}</span>
        </div>
        <div class="kpi-card">
          <span class="kpi-label">Suppliers</span>
          <span class="kpi-value">{{ stats.suppliers.toLocaleString() }}</span>
        </div>
      </div>

      <!-- Quick Order -->
      <div class="quick-order-panel">
        <div class="quick-order-header">
          <span class="material-symbols-outlined quick-order-icon">bolt</span>
          <span class="quick-order-title">Quick-Order SKU</span>
        </div>
        <p class="quick-order-desc">
          Enter an enterprise SKU or manufacturer part number for instant requisition staging.
        </p>
        <div class="quick-order-form">
          <input
            v-model="skuInput"
            type="text"
            class="input quick-order-input"
            placeholder="e.g. PRD-VLV-9021"
          />
          <input v-model.number="qtyInput" type="number" min="1" class="input quick-order-qty" />
          <button class="button quick-order-btn" @click="stageItem">
            <span class="material-symbols-outlined">add_shopping_cart</span>
            Stage
          </button>
        </div>
        <div v-if="stagedCount > 0" class="quick-order-feedback">
          <span class="material-symbols-outlined">inventory_2</span>
          <span>{{ stagedCount }} item{{ stagedCount > 1 ? 's' : '' }} staged</span>
          <a href="/cart" class="quick-order-review">Review &rarr;</a>
        </div>
        <div class="quick-order-presets">
          <button class="preset-btn" @click="presetSku('VLV-IND-9021')">VLV-IND-9021</button>
          <button class="preset-btn" @click="presetSku('MTR-EL-4402')">MTR-EL-4402</button>
          <button class="preset-btn" @click="presetSku('CBL-NET-8831')">CBL-NET-8831</button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.hero-section {
  margin-bottom: var(--space-xl);
}

.hero-status-bar {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-xs) var(--space-md);
  background-color: var(--surface-container-low);
  border-radius: var(--radius-lg);
  margin-bottom: var(--space-lg);
}

.hero-status-icon {
  font-size: 16px;
  color: var(--secondary);
}

.hero-status-text {
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.hero-status-sep {
  color: var(--outline-variant);
}

.hero-grid {
  display: grid;
  grid-template-columns: 1fr 440px;
  gap: var(--space-xl);
  align-items: start;
}

/* ── KPIs ── */
.hero-kpis {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--space-md);
}

.kpi-card {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.kpi-label {
  font-size: var(--text-label-sm);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: var(--tracking-label-sm);
  color: var(--muted);
}

.kpi-value {
  font-size: var(--text-tabular-lg);
  font-weight: 600;
  color: var(--on-surface);
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  line-height: var(--line-tabular-lg);
  font-variant-numeric: tabular-nums;
}

.kpi-check {
  font-size: 18px;
  color: var(--status-ok);
}

.kpi-change {
  font-size: var(--text-label-sm);
  font-weight: 500;
}

.kpi-change-up {
  color: var(--status-ok);
}

.kpi-change-warn {
  color: var(--status-warning);
}

/* ── Quick Order ── */
.quick-order-panel {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.quick-order-header {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.quick-order-icon {
  font-size: 20px;
  color: var(--secondary);
}

.quick-order-title {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
}

.quick-order-desc {
  margin: 0;
  font-size: var(--text-body-sm);
  color: var(--muted);
}

.quick-order-form {
  display: flex;
  gap: var(--space-sm);
}

.quick-order-input {
  flex: 1;
  min-width: 0;
}

.quick-order-qty {
  width: 72px;
  text-align: center;
}

.quick-order-btn {
  flex-shrink: 0;
}

.quick-order-btn .material-symbols-outlined {
  font-size: 18px;
}

.quick-order-feedback {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-sm) var(--space-md);
  background-color: var(--surface-container-low);
  border-radius: var(--radius);
  font-size: var(--text-label-md);
  color: var(--on-surface);
  font-weight: 500;
}

.quick-order-feedback .material-symbols-outlined {
  font-size: 18px;
  color: var(--secondary);
}

.quick-order-review {
  margin-left: auto;
  color: var(--secondary);
  font-weight: 600;
  text-decoration: none;
}

.quick-order-review:hover {
  color: var(--accent-hover);
}

.quick-order-presets {
  display: flex;
  gap: var(--space-xs);
}

.preset-btn {
  padding: var(--space-xs) var(--space-sm);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface-container-low);
  font-family: var(--font-family);
  font-size: var(--text-label-sm);
  font-variant-numeric: tabular-nums;
  color: var(--muted);
  cursor: pointer;
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
}

.preset-btn:hover {
  background-color: var(--surface-container-high);
  color: var(--on-surface);
}

@media (max-width: 1024px) {
  .hero-grid {
    grid-template-columns: 1fr;
  }
}
</style>
