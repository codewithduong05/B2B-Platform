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

const activeTab = ref('dimensions')
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
        <!-- Left: Images, Artifacts, Specs -->
        <div class="product-left">
          <ProductImageGallery :images="product.images" />
          <ProductEngineeringArtifacts :artifacts="product.artifacts" />
          <ProductSpecsGrid :specs="product.specs" standard="DIN EN 60529" />
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

            <ProductPriceCalculation
              :unit-price="currentUnitPrice"
              :tiers="product.pricing.tiers"
              :unit="product.pricing.unit"
              :active-tier-index="activeTierIndex"
              :qty="selectedQty"
              :subtotal="subtotal"
              @update:qty="(v: number) => setQty(v)"
              @adjust="adjustQty"
              @set="setQty"
            />

            <ProductStockRadar :locations="product.stock" :total-stock="product.totalStock" />

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

          <ProductSupplierCard :supplier="product.supplier" />
        </div>
      </div>

      <!-- Bottom Tabs Section -->
      <div class="product-docs-section">
        <div class="product-docs-header">
          <div>
            <h2 class="product-docs-title">Comprehensive Engineering Documentation</h2>
            <p class="product-docs-desc">
              Procurement specifications, manifold drill patterns, and life-cycle reliability data.
            </p>
          </div>
          <div class="product-docs-tabs">
            <button
              class="tab"
              :class="{ 'tab-active': activeTab === 'dimensions' }"
              @click="activeTab = 'dimensions'"
            >
              Dimensional Layout
            </button>
            <button
              class="tab"
              :class="{ 'tab-active': activeTab === 'wiring' }"
              @click="activeTab = 'wiring'"
            >
              Wiring &amp; Control
            </button>
            <button
              class="tab"
              :class="{ 'tab-active': activeTab === 'maintenance' }"
              @click="activeTab = 'maintenance'"
            >
              Maintenance Schedules
            </button>
          </div>
        </div>
        <div class="product-docs-content">
          <div class="product-docs-blueprint">
            <span class="material-symbols-outlined product-docs-blueprint-icon">engineering</span>
            <span class="product-docs-blueprint-badge">Tolerance ±0.05mm</span>
          </div>
          <div class="product-docs-cards">
            <div class="product-doc-card">
              <h4 class="product-doc-card-title">Coil Excitation Specs</h4>
              <p class="product-doc-card-body">
                Standard 24V DC excitation allows continuous energization without auxiliary chilling
                up to an ambient ceiling of 65°C. Transient suppressor diodes built-in to nullify
                inductive flyback voltage.
              </p>
            </div>
            <div class="product-doc-card">
              <h4 class="product-doc-card-title">Elastomer Sealing Chemistry</h4>
              <p class="product-doc-card-body">
                FKM (Viton®) elastomeric seals provide impervious barrier resistance to synthetic
                compressor oils, minor solvent vapors, and volatile hydrocarbons down to -10°C.
              </p>
            </div>
            <div class="product-doc-card product-doc-card-cta">
              <div class="product-doc-cta-left">
                <span class="material-symbols-outlined product-doc-cta-icon">support_agent</span>
                <div>
                  <span class="product-doc-cta-title"
                    >Need custom manifold spacing or hazardous zone ATEX cert?</span
                  >
                  <span class="product-doc-cta-desc"
                    >{{ product?.supplier?.name || 'Supplier' }} OEM Field Application Engineers available for Atlas buyers.</span
                  >
                </div>
              </div>
              <button class="button button-secondary">Contact Applications Engineer</button>
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

/* ── Docs Section ── */
.product-docs-section {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: var(--space-xl);
  margin-top: var(--space-xl);
}

.product-docs-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  border-bottom: 1px solid var(--surface-container-high);
  padding-bottom: var(--space-md);
  margin-bottom: var(--space-lg);
  flex-wrap: wrap;
  gap: var(--space-md);
}

.product-docs-title {
  font-size: var(--text-headline-md);
  font-weight: 700;
  color: var(--on-surface);
  margin: 0;
}

.product-docs-desc {
  margin: 4px 0 0;
  font-size: var(--text-body-sm);
  color: var(--muted);
}

.product-docs-tabs {
  display: flex;
  gap: var(--space-xs);
  background-color: var(--surface-container-low);
  padding: 4px;
  border-radius: var(--radius-lg);
}

.product-docs-content {
  display: grid;
  grid-template-columns: 1fr 2fr;
  gap: var(--space-xl);
  align-items: center;
}

.product-docs-blueprint {
  aspect-ratio: 4 / 3;
  border-radius: var(--radius-lg);
  background-color: var(--surface-container-low);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.product-docs-blueprint-icon {
  font-size: 48px;
  color: var(--outline-variant);
}

.product-docs-blueprint-badge {
  position: absolute;
  bottom: 8px;
  left: 8px;
  padding: 2px 8px;
  border-radius: var(--radius);
  background-color: rgba(0, 0, 0, 0.8);
  color: var(--on-primary);
  font-family: monospace;
  font-size: 10px;
}

.product-docs-cards {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.product-doc-card {
  padding: var(--space-md);
  border-radius: var(--radius-lg);
  background-color: var(--surface-container-low);
}

.product-doc-card-title {
  font-size: var(--text-label-md);
  font-weight: 700;
  color: var(--on-surface);
  margin: 0 0 var(--space-xs);
}

.product-doc-card-body {
  margin: 0;
  font-size: var(--text-body-sm);
  color: var(--muted);
  line-height: 1.6;
}

.product-doc-card-cta {
  background-color: var(--surface-container);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--space-md);
}

.product-doc-cta-left {
  display: flex;
  align-items: flex-start;
  gap: var(--space-sm);
}

.product-doc-cta-icon {
  font-size: 24px;
  color: var(--secondary);
  flex-shrink: 0;
}

.product-doc-cta-title {
  display: block;
  font-size: var(--text-label-md);
  font-weight: 700;
  color: var(--on-surface);
}

.product-doc-cta-desc {
  display: block;
  font-size: var(--text-body-sm);
  color: var(--muted);
}

@media (max-width: 1024px) {
  .product-layout {
    grid-template-columns: 1fr;
  }

  .product-docs-content {
    grid-template-columns: 1fr;
  }
}
</style>
