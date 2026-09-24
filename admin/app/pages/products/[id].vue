<template>
  <div class="space-y-6">
    <nav class="breadcrumb">
      <NuxtLink to="/products" class="breadcrumb-link">Products</NuxtLink>
      <span class="breadcrumb-sep">/</span>
      <span class="breadcrumb-current">{{ id }}</span>
    </nav>

    <div v-if="loading" class="card">Loading product...</div>
    <div v-else-if="error" class="card error-text">{{ error }}</div>
    <template v-else-if="product">
      <div class="flex items-start justify-between">
        <div>
          <h1 class="text-2xl font-bold">{{ product.name }}</h1>
          <p class="text-sm text-muted mt-1">SKU: {{ product.code || product.sku }}</p>
        </div>
        <span class="badge" :class="`badge--${product.status}`">{{ product.status }}</span>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div class="card">
          <h2 class="card-title">Details</h2>
          <div class="detail-grid">
            <div class="detail-row">
              <span class="detail-label">Category</span>
              <span>{{ product.category_name || '—' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Brand</span>
              <span>{{ product.brand_name || '—' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Supplier</span>
              <span>{{ product.supplier_name || '—' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Handling Class</span>
              <span>{{ product.handling_class || '—' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Base Price</span>
              <span>{{ product.base_price_minor != null ? `$${(product.base_price_minor / 100).toFixed(2)}` : '—' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Currency</span>
              <span>{{ product.currency || '—' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Track Inventory</span>
              <span>{{ product.track_inventory ? 'Yes' : 'No' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Active</span>
              <span>{{ product.is_active ? 'Yes' : 'No' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Featured</span>
              <span>{{ product.is_featured ? 'Yes' : 'No' }}</span>
            </div>
          </div>
        </div>

        <div class="card">
          <h2 class="card-title">Physical</h2>
          <div class="detail-grid">
            <div class="detail-row">
              <span class="detail-label">Weight</span>
              <span>{{ product.weight_grams ? `${product.weight_grams}g` : '—' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Dimensions</span>
              <span>{{ product.length_mm && product.width_mm && product.height_mm ? `${product.length_mm} × ${product.width_mm} × ${product.height_mm} mm` : '—' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">GTIN</span>
              <span class="mono">{{ product.gtin || '—' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Created</span>
              <span>{{ product.created_at ? new Date(product.created_at).toLocaleDateString() : '—' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Updated</span>
              <span>{{ product.updated_at ? new Date(product.updated_at).toLocaleDateString() : '—' }}</span>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const id = route.params.id as string

const product = ref<any>(null)
const loading = ref(true)
const error = ref('')

async function loadProduct() {
  loading.value = true
  try {
    product.value = await $fetch(`/api/catalog/products/${id}`)
  } catch (e: any) {
    error.value = e?.data?.message || 'Product not found'
  } finally {
    loading.value = false
  }
}

onMounted(loadProduct)
</script>

<style scoped>
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.75rem;
  color: var(--muted);
}

.breadcrumb-link { color: var(--muted); text-decoration: none; }
.breadcrumb-link:hover { color: var(--secondary); }
.breadcrumb-sep { color: var(--outline-variant); }
.breadcrumb-current { font-weight: 600; color: var(--text); }

.card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
}

.card-title {
  font-size: 1rem;
  font-weight: 600;
  margin-bottom: 1rem;
}

.detail-grid {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  font-size: 0.875rem;
}

.detail-label {
  color: var(--muted);
  font-weight: 500;
}

.mono { font-family: monospace; }

.error-text { color: var(--error); }

.badge {
  padding: 4px 12px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: capitalize;
}

.badge--active { background: #dcfce7; color: #16a34a; }
.badge--draft { background: var(--surface-container-low); color: var(--muted); }
.badge--archived { background: #fee2e2; color: #dc2626; }
</style>
