<script setup lang="ts">
import type { CatalogProduct } from '~~/shared/catalog'

defineProps<{ product: CatalogProduct }>()

const qty = ref(1)
</script>

<template>
  <article class="product-card">
    <div class="product-card-top">
      <div class="product-card-badges">
        <span class="product-card-supplier">{{ product.supplier }}</span>
        <span class="product-card-sku">SKU: {{ product.sku }}</span>
      </div>
      <div class="product-card-img">
        <span class="material-symbols-outlined product-card-img-icon">inventory_2</span>
        <span class="product-card-img-overlay">{{ product.specs }}</span>
      </div>
      <h3 class="product-card-name">{{ product.name }}</h3>
      <p class="product-card-desc">{{ product.description }}</p>
      <div class="product-card-stock" :class="{ 'product-card-stock-low': product.lowStock }">
        <span class="product-card-stock-dot"></span>
        <span class="product-card-stock-info">{{ product.stockLabel }}</span>
        <span class="product-card-stock-warehouse">{{ product.warehouse }}</span>
      </div>
    </div>
    <div class="product-card-pricing">
      <div class="product-card-price">
        <span class="product-card-price-label">Enterprise Price</span>
        <div class="product-card-price-row">
          <span class="product-card-price-value">${{ product.price.toFixed(2) }}</span>
          <span class="product-card-price-unit">/ {{ product.unit }}</span>
        </div>
      </div>
      <div v-if="product.bulkPrice" class="product-card-bulk">
        <span class="product-card-bulk-price">${{ product.bulkPrice.toFixed(2) }}</span>
        <span class="product-card-bulk-label">{{ product.bulkLabel }}</span>
      </div>
    </div>
    <div class="product-card-actions">
      <div class="product-card-qty">
        <button class="qty-btn" @click="qty = Math.max(1, qty - 1)">-</button>
        <input v-model.number="qty" type="number" :min="product.moq" class="qty-input" />
        <button class="qty-btn" @click="qty++">+</button>
      </div>
      <NuxtLink :to="`/product/${product.sku}`" class="button product-card-requisition">
        <span class="material-symbols-outlined">shopping_cart</span>
        Requisition
      </NuxtLink>
    </div>
  </article>
</template>

<style scoped>
.product-card {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-md);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: var(--space-md);
  transition: box-shadow 0.15s ease;
}

.product-card:hover {
  box-shadow: var(--shadow-2);
}

.product-card-top {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.product-card-badges {
  display: flex;
  align-items: start;
  justify-content: space-between;
  gap: var(--space-xs);
}

.product-card-supplier {
  padding: 2px 8px;
  border-radius: var(--radius);
  background-color: var(--surface-container-high);
  font-size: var(--text-label-sm);
  font-weight: 600;
  color: var(--secondary);
  text-transform: uppercase;
}

.product-card-sku {
  padding: 2px 8px;
  border-radius: var(--radius);
  background-color: var(--surface);
  font-size: var(--text-tabular-sm);
  font-weight: 700;
  color: var(--on-surface);
  font-variant-numeric: tabular-nums;
}

.product-card-img {
  width: 100%;
  height: 160px;
  background-color: var(--surface-container-low);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.product-card-img-icon {
  font-size: 40px;
  color: var(--outline-variant);
}

.product-card-img-overlay {
  position: absolute;
  bottom: 8px;
  left: 8px;
  padding: 2px 6px;
  border-radius: var(--radius);
  background-color: rgba(0, 0, 0, 0.8);
  color: var(--on-primary);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.product-card-name {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
  margin: 0;
}

.product-card-desc {
  margin: 0;
  font-size: var(--text-body-sm);
  color: var(--muted);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.product-card-stock {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-sm);
  border-radius: var(--radius);
  background-color: var(--surface-container-low);
}

.product-card-stock-low {
  background-color: var(--status-critical-bg);
}

.product-card-stock-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  background-color: var(--secondary);
  flex-shrink: 0;
}

.product-card-stock-low .product-card-stock-dot {
  background-color: var(--error);
}

.product-card-stock-info {
  font-size: var(--text-label-sm);
  font-weight: 600;
  color: var(--on-surface);
}

.product-card-stock-low .product-card-stock-info {
  color: var(--error);
}

.product-card-stock-warehouse {
  font-size: var(--text-label-sm);
  color: var(--muted);
  margin-left: auto;
}

/* ── Pricing ── */
.product-card-pricing {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}

.product-card-price {
  display: flex;
  flex-direction: column;
}

.product-card-price-label {
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.product-card-price-row {
  display: flex;
  align-items: baseline;
  gap: var(--space-xs);
}

.product-card-price-value {
  font-size: var(--text-tabular-lg);
  font-weight: 700;
  color: var(--on-surface);
  font-variant-numeric: tabular-nums;
}

.product-card-price-unit {
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.product-card-bulk {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.product-card-bulk-price {
  font-size: var(--text-label-md);
  font-weight: 700;
  color: var(--secondary);
}

.product-card-bulk-label {
  font-size: var(--text-label-sm);
  color: var(--muted);
}

/* ── Actions ── */
.product-card-actions {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding-top: var(--space-xs);
  border-top: 1px solid var(--border);
}

.product-card-qty {
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

.product-card-requisition {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-xs);
  height: 40px;
  padding: 0 var(--space-md);
  border: none;
  border-radius: var(--radius);
  background-color: var(--accent);
  color: var(--on-primary);
  font-family: var(--font-family);
  font-size: var(--text-label-md);
  font-weight: 600;
  text-decoration: none;
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.product-card-requisition:hover {
  background-color: var(--accent-hover);
}

.product-card-requisition .material-symbols-outlined {
  font-size: 16px;
}
</style>
