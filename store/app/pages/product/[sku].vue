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
    <div v-if="loading" class="product-loading">Loading product details...</div>

    <div v-else-if="error" class="product-error">{{ error }}</div>

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
        <span class="product-handling-badge">
          <span class="material-symbols-outlined">thermostat</span>
          {{ product.handling_class }}
        </span>
        <span v-for="cert in product.certifications" :key="cert" class="product-cert-badge">
          <span class="material-symbols-outlined">verified</span>
          {{ cert }}
        </span>
        <span v-if="product.unspsc" class="product-unspsc-badge">UNSPSC {{ product.unspsc }}</span>
      </div>

      <!-- 2-Column Layout -->
      <div class="product-layout">
        <!-- Left Column: Image + Specs + Description -->
        <div class="product-left">
          <ProductImageGallery :images="product.images" />

          <ProductSpecsGrid :specs="product.specs" />

          <div v-if="product.description" class="product-description">
            <h3 class="product-description-title">Description</h3>
            <p class="product-description-body">{{ product.description }}</p>
          </div>

          <ProductEngineeringArtifacts
            v-if="product.artifacts.length"
            :artifacts="product.artifacts"
          />
        </div>

        <!-- Right Column: Purchase Command -->
        <div class="product-right">
          <!-- Identity Header -->
          <div class="product-identity">
            <div class="product-identity-meta">
              <span v-if="product.supplier.name" class="product-partner-badge">
                <span class="material-symbols-outlined">verified</span>
                {{ product.supplier.name }} Partner
              </span>
              <span class="product-mpn">MPN: {{ product.mpn }}</span>
            </div>
            <h1 class="product-title">{{ product.name }}</h1>
            <div class="product-sku-line">
              <span>Sku: <strong>{{ product.sku }}</strong></span>
            </div>
          </div>

          <!-- Pricing Card -->
          <div class="product-purchase-card">
            <ProductPriceCalculation
              :unit-price="currentUnitPrice"
              :tiers="product.pricing.tiers"
              :unit="product.pricing.unit"
              :active-tier-index="activeTierIndex"
              :qty="selectedQty"
              :subtotal="subtotal"
              @update:qty="setQty"
              @adjust="adjustQty"
            />

            <!-- CTAs -->
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

          <!-- Stock Radar -->
          <ProductStockRadar
            v-if="product.stock.length"
            :locations="product.stock"
            :total-stock="product.totalStock"
          />

          <!-- Supplier Card -->
          <ProductSupplierCard
            v-if="product.supplier.name"
            :supplier="product.supplier"
          />
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
  flex-wrap: wrap;
}

.product-handling-badge {
  display: inline-flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-sm) var(--space-md);
  border-radius: var(--radius);
  background-color: var(--tertiary-fixed);
  font-size: var(--text-label-sm);
  font-weight: 600;
  color: var(--tertiary);
  text-transform: uppercase;
}

.product-handling-badge .material-symbols-outlined {
  font-size: 14px;
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

/* ── Identity ── */
.product-identity {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.product-identity-meta {
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

@media (max-width: 1024px) {
  .product-layout {
    grid-template-columns: 1fr;
  }
}
</style>
