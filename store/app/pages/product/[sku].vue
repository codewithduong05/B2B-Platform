<script setup lang="ts">
import { useProduct } from '~/composables/useProduct'

const route = useRoute()
const sku = route.params.sku as string
const { addToCart } = useCart()

const {
  product,
  loading,
  error,
  selectedQty,
  activeTierIndex,
  currentUnitPrice,
  subtotal,
  fetchProduct,
  adjustQty,
  setQty,
} = useProduct(sku)

const addingToCart = ref(false)

async function handleAddToCart() {
  if (!product.value) return
  addingToCart.value = true
  try {
    await addToCart(product.value.sku, selectedQty.value)
  } catch {
    // Cart requires auth
  } finally {
    addingToCart.value = false
  }
}

onMounted(() => {
  fetchProduct()
})
</script>

<template>
  <div class="product-page">
    <!-- Loading -->
    <div v-if="loading" class="product-loading">Loading product details...</div>

    <!-- Error -->
    <div v-else-if="error" class="product-error">{{ error }}</div>

    <!-- Content -->
    <template v-else-if="product">
      <!-- Breadcrumb -->
      <nav class="product-breadcrumb">
        <span v-for="(crumb, i) in product.category" :key="i" class="product-breadcrumb-item">
          <a v-if="i < product.category.length - 1" href="/catalog" class="product-breadcrumb-link">
            {{ crumb }}
          </a>
          <span v-else class="product-breadcrumb-current">{{ crumb }}</span>
          <span v-if="i < product.category.length - 1" class="product-breadcrumb-sep">
            <span class="material-symbols-outlined">chevron_right</span>
          </span>
        </span>
        <span class="product-breadcrumb-sep">
          <span class="material-symbols-outlined">chevron_right</span>
        </span>
        <span class="product-breadcrumb-sku">SKU: {{ product.sku }}</span>
      </nav>

      <!-- Classification Badges -->
      <div class="product-badges">
        <span v-for="cert in product.certifications" :key="cert" class="product-cert-badge">
          <span class="material-symbols-outlined">verified</span>
          {{ cert }}
        </span>
        <span class="product-unspsc-badge">UNSPSC {{ product.unspsc }}</span>
      </div>

      <!-- 2-Column Layout -->
      <div class="product-layout">
        <!-- Left: Product Info -->
        <div class="product-left">
          <div class="product-specs">
            <h3 class="product-specs-title">Specifications</h3>
            <div v-if="product.specs.length" class="product-specs-grid">
              <template v-for="spec in product.specs" :key="spec.label">
                <div class="product-spec-label" :class="{ 'product-spec-highlight': spec.highlight }">
                  {{ spec.label }}
                </div>
                <div class="product-spec-value" :class="{ 'product-spec-highlight': spec.highlight }">
                  {{ spec.value }}
                </div>
              </template>
            </div>
            <p v-else class="product-specs-empty">No specifications available.</p>
          </div>

          <div v-if="product.description" class="product-description">
            <h3 class="product-description-title">Description</h3>
            <p class="product-description-body">{{ product.description }}</p>
          </div>
        </div>

        <!-- Right: Purchasing Command -->
        <div class="product-right">
          <!-- Purchasing Card -->
          <div class="product-purchase-card">
            <!-- Header Metadata -->
            <div class="product-purchase-header">
              <div class="product-purchase-meta">
                <span class="product-partner-badge">
                  <span class="material-symbols-outlined">verified</span>
                  {{ product.supplier.name }} Partner
                </span>
                <span class="product-mpn">MPN: {{ product.mpn }}</span>
              </div>
              <h1 class="product-title">{{ product.name }}</h1>
              <div class="product-sku-line">
                <span
                  >SKU: <strong>{{ product.sku }}</strong></span
                >
                <span class="product-sku-sep">&bull;</span>
                <span class="product-msa">In Master Service Agreement</span>
              </div>
            </div>

            <div class="product-price-section">
              <div class="product-price-row">
                <span class="product-price-label">Unit Price</span>
                <span class="product-price-value">${{ currentUnitPrice.toFixed(2) }} / {{ product.pricing.unit }}</span>
              </div>
              <div v-if="product.pricing.tiers.length > 1" class="product-tiers">
                <div
                  v-for="(tier, i) in product.pricing.tiers"
                  :key="tier.id"
                  class="product-tier"
                  :class="{ 'product-tier-active': i === activeTierIndex }"
                >
                  <span class="product-tier-range">{{ tier.range }}</span>
                  <span class="product-tier-price">${{ tier.price.toFixed(2) }}</span>
                  <span class="product-tier-discount">{{ tier.discount }}</span>
                </div>
              </div>
              <div class="product-qty-row">
                <label class="product-qty-label">Quantity</label>
                <div class="product-qty-controls">
                  <button class="button button-sm" @click="adjustQty(-1)">-</button>
                  <input
                    type="number"
                    class="product-qty-input"
                    :value="selectedQty"
                    min="1"
                    @change="(e: Event) => setQty(Number((e.target as HTMLInputElement).value) || 1)"
                  />
                  <button class="button button-sm" @click="adjustQty(1)">+</button>
                </div>
              </div>
              <div class="product-subtotal-row">
                <span class="product-subtotal-label">Subtotal</span>
                <span class="product-subtotal-value">${{ subtotal.toFixed(2) }}</span>
              </div>
            </div>

            <div v-if="product.stock.length" class="product-stock-section">
              <h4 class="product-stock-title">Availability</h4>
              <div v-for="loc in product.stock" :key="loc.warehouse" class="product-stock-row">
                <span class="product-stock-warehouse">{{ loc.warehouse }}</span>
                <span class="product-stock-units" :class="{ 'product-stock-unavailable': loc.units === 0 }">
                  {{ loc.units }} units
                </span>
              </div>
            </div>

            <!-- Action CTAs -->
            <div class="product-ctas">
              <button class="button product-cta-primary" :disabled="addingToCart" @click="handleAddToCart">
                <span class="material-symbols-outlined">shopping_cart</span>
                {{ addingToCart ? 'Adding...' : 'Add to Staging Cart' }}
              </button>
              <button class="button button-secondary product-cta-secondary">
                <span class="material-symbols-outlined">request_quote</span>
                Request Formal Volume Quote (RFQ)
              </button>
            </div>
          </div>

          <div class="product-supplier-card">
            <h3 class="product-supplier-title">Supplier</h3>
            <div class="product-supplier-info">
              <span class="product-supplier-name">{{ product.supplier.name }}</span>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.product-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

.product-loading,
.product-error {
  padding: var(--space-2xl);
  text-align: center;
  font-size: var(--text-body-md);
  color: var(--muted);
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.product-error {
  color: var(--error);
  background-color: var(--error-bg);
}

/* ── Breadcrumb ── */
.product-breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-label-sm);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: var(--tracking-label-sm);
  color: var(--muted);
  flex-wrap: wrap;
}

.product-breadcrumb-link {
  color: var(--muted);
  text-decoration: none;
  transition: color 0.15s ease;
}

.product-breadcrumb-link:hover {
  color: var(--secondary);
}

.product-breadcrumb-sep {
  display: flex;
}

.product-breadcrumb-sep .material-symbols-outlined {
  font-size: 14px;
  color: var(--outline-variant);
}

.product-breadcrumb-current {
  color: var(--on-surface);
}

.product-breadcrumb-sku {
  padding: 2px 8px;
  border-radius: var(--radius);
  background-color: var(--surface-container-high);
  color: var(--on-surface);
}

/* ── Badges ── */
.product-badges {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.product-cert-badge {
  display: inline-flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-sm) var(--space-md);
  border-radius: var(--radius);
  background-color: var(--surface-container-high);
  font-size: var(--text-label-sm);
  font-weight: 600;
  color: var(--secondary);
  text-transform: uppercase;
}

.product-cert-badge .material-symbols-outlined {
  font-size: 14px;
}

.product-unspsc-badge {
  padding: var(--space-sm) var(--space-md);
  border-radius: var(--radius);
  background-color: var(--surface-container);
  font-size: var(--text-label-sm);
  color: var(--muted);
}

/* ── 2-Column Layout ── */
.product-layout {
  display: grid;
  grid-template-columns: 7fr 5fr;
  gap: var(--space-xl);
  align-items: start;
}

.product-left {
  display: flex;
  flex-direction: column;
  gap: var(--space-xl);
}

.product-right {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

/* ── Purchase Card ── */
.product-purchase-card {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.product-purchase-header {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.product-purchase-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.product-partner-badge {
  display: inline-flex;
  align-items: center;
  gap: var(--space-xs);
  padding: 2px 8px;
  border-radius: var(--radius);
  background-color: var(--secondary-fixed);
  font-size: var(--text-label-sm);
  font-weight: 600;
  color: var(--on-secondary-fixed);
}

.product-partner-badge .material-symbols-outlined {
  font-size: 13px;
}

.product-mpn {
  font-size: var(--text-label-sm);
  color: var(--muted);
  font-family: monospace;
}

.product-title {
  font-size: var(--text-headline-lg);
  font-weight: 700;
  color: var(--on-surface);
  letter-spacing: var(--tracking-headline-lg);
  margin: 0;
}

.product-sku-line {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  font-size: var(--text-label-md);
  color: var(--muted);
}

.product-sku-line strong {
  color: var(--on-surface);
}

.product-sku-sep {
  color: var(--outline-variant);
}

.product-msa {
  color: var(--secondary);
  font-weight: 500;
}

/* ── CTAs ── */
.product-ctas {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.product-cta-primary {
  width: 100%;
  height: 44px;
  justify-content: center;
  gap: var(--space-sm);
  font-size: var(--text-headline-sm);
  border: none;
}

.product-cta-primary .material-symbols-outlined {
  font-size: 20px;
}

.product-cta-secondary {
  width: 100%;
  height: 40px;
  justify-content: center;
  gap: var(--space-xs);
}

.product-cta-secondary .material-symbols-outlined {
  font-size: 18px;
  color: var(--secondary);
}

/* ── Specs ── */
.product-specs {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: var(--space-lg);
}

.product-specs-title {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
  margin: 0 0 var(--space-md);
}

.product-specs-grid {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: var(--space-sm) var(--space-lg);
}

.product-spec-label {
  font-size: var(--text-label-sm);
  font-weight: 600;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: var(--tracking-label-sm);
}

.product-spec-value {
  font-size: var(--text-body-sm);
  color: var(--on-surface);
}

.product-spec-highlight {
  color: var(--secondary);
}

.product-specs-empty {
  font-size: var(--text-body-sm);
  color: var(--muted);
}

/* ── Description ── */
.product-description {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: var(--space-lg);
}

.product-description-title {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
  margin: 0 0 var(--space-md);
}

.product-description-body {
  margin: 0;
  font-size: var(--text-body-md);
  color: var(--muted);
  line-height: 1.6;
}

/* ── Price Section ── */
.product-price-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  padding: var(--space-md) 0;
  border-bottom: 1px solid var(--surface-container-high);
}

.product-price-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}

.product-price-label {
  font-size: var(--text-label-md);
  color: var(--muted);
}

.product-price-value {
  font-size: var(--text-headline-md);
  font-weight: 700;
  color: var(--on-surface);
}

.product-tiers {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.product-tier {
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: var(--space-sm);
  padding: var(--space-xs) var(--space-sm);
  border-radius: var(--radius);
  font-size: var(--text-body-sm);
  color: var(--muted);
}

.product-tier-active {
  background-color: var(--secondary-fixed);
  color: var(--on-surface);
}

.product-tier-range {
  font-weight: 600;
}

.product-tier-price {
  font-weight: 600;
  color: var(--on-surface);
}

.product-tier-discount {
  color: var(--secondary);
  font-size: var(--text-label-sm);
}

.product-qty-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-md);
}

.product-qty-label {
  font-size: var(--text-label-md);
  color: var(--muted);
}

.product-qty-controls {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
}

.product-qty-input {
  width: 56px;
  height: 36px;
  text-align: center;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  font-size: var(--text-body-md);
}

.product-subtotal-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding-top: var(--space-sm);
  border-top: 1px solid var(--surface-container-high);
}

.product-subtotal-label {
  font-size: var(--text-label-md);
  font-weight: 600;
  color: var(--on-surface);
}

.product-subtotal-value {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
}

/* ── Stock Section ── */
.product-stock-section {
  padding: var(--space-md) 0;
}

.product-stock-title {
  font-size: var(--text-label-md);
  font-weight: 600;
  color: var(--on-surface);
  margin: 0 0 var(--space-sm);
}

.product-stock-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-xs) 0;
  font-size: var(--text-body-sm);
}

.product-stock-warehouse {
  color: var(--muted);
}

.product-stock-units {
  font-weight: 600;
  color: var(--on-surface);
}

.product-stock-unavailable {
  color: var(--error);
}

/* ── Supplier Card ── */
.product-supplier-card {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: var(--space-lg);
}

.product-supplier-title {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
  margin: 0 0 var(--space-md);
}

.product-supplier-name {
  font-size: var(--text-body-md);
  font-weight: 600;
  color: var(--on-surface);
}

@media (max-width: 1024px) {
  .product-layout {
    grid-template-columns: 1fr;
  }
}
</style>
