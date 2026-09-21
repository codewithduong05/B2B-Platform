<script setup lang="ts">
import { useCatalog } from '~/composables/useCatalog'

definePageMeta({ title: 'Catalog' })

const {
  state,
  loading,
  error,
  activeFilters,
  totalPages,
  paginatedRange,
  fetchProducts,
  setPage,
  setSort,
  setViewMode,
  removeFilter,
  applyFilters,
} = useCatalog()

const categoryInfo = ref({
  unspsc: '',
  title: 'Product Catalog',
  description:
    'Browse and search the full product catalog. Use filters to narrow results by supplier, category, and availability.',
  sla: '',
})

const catalogCount = computed(() => state.value.total || 0)

onMounted(() => {
  fetchProducts()
})

function handleFilterUpdate(filters: Record<string, unknown>) {
  applyFilters(filters as Record<string, never>)
}

function handleFilterReset() {
  applyFilters({
    suppliers: [],
    leadTime: '',
    warehouses: [],
    moqMax: 100,
    compliance: [],
  })
}
</script>

<template>
  <div class="catalog-page">
    <!-- Breadcrumb -->
    <nav class="catalog-breadcrumb">
      <a href="/" class="catalog-breadcrumb-link">Home</a>
      <span class="catalog-breadcrumb-sep">/</span>
      <span class="catalog-breadcrumb-current">Catalog</span>
    </nav>

    <!-- Header + Metric -->
    <div class="catalog-header-grid">
      <div class="catalog-header-panel">
        <div class="catalog-header-badges">
          <span v-if="categoryInfo.unspsc" class="catalog-header-unspsc">UNSPSC {{ categoryInfo.unspsc }}</span>
          <span class="catalog-header-count">{{ catalogCount }} Products</span>
        </div>
        <h1 class="catalog-header-title">{{ categoryInfo.title }}</h1>
        <p class="catalog-header-desc">{{ categoryInfo.description }}</p>
        <div class="catalog-header-actions">
          <button class="button button-secondary">
            <span class="material-symbols-outlined">file_download</span>
            Export CSV
          </button>
          <button class="button">
            <span class="material-symbols-outlined">assignment_add</span>
            Batch RFQ
          </button>
        </div>
      </div>
    </div>

    <!-- Main Content: Sidebar + Catalog -->
    <div class="catalog-content-grid">
      <CatalogFilterSidebar
        :filters="state.filters"
        @update:filters="handleFilterUpdate"
        @reset="handleFilterReset"
      />

      <section class="catalog-main">
        <CatalogControlBar
          :active-filters="activeFilters"
          :sort-by="state.sortBy"
          :view-mode="state.viewMode"
          @remove-filter="removeFilter"
          @sort="setSort"
          @view-mode="setViewMode"
        />

        <div v-if="loading" class="catalog-loading">Loading products...</div>
        <div v-else-if="error" class="catalog-error">{{ error }}</div>
        <template v-else>
          <CatalogProductGrid v-if="state.viewMode === 'grid'" :products="state.products" />
          <CatalogProductTable v-else :products="state.products" />
        </template>

        <CatalogPagination
          :current-page="state.page"
          :total-pages="totalPages"
          :total="paginatedRange.total"
          :per-page="state.perPage"
          :start="paginatedRange.start"
          :end="paginatedRange.end"
          @page="setPage"
        />
      </section>
    </div>
  </div>
</template>

<style scoped>
.catalog-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

/* ── Breadcrumb ── */
.catalog-breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-xs) var(--space-md);
  background-color: var(--surface-container-low);
  border-radius: var(--radius-lg);
  font-size: var(--text-label-md);
  color: var(--muted);
  flex-wrap: wrap;
}

.catalog-breadcrumb-link {
  color: var(--muted);
  text-decoration: none;
  transition: color 0.15s ease;
}

.catalog-breadcrumb-link:hover {
  color: var(--secondary);
}

.catalog-breadcrumb-sep {
  color: var(--outline-variant);
}

.catalog-breadcrumb-current {
  font-weight: 600;
  color: var(--on-surface);
}

/* ── Header Grid ── */
.catalog-header-grid {
  display: grid;
  grid-template-columns: 3fr 1fr;
  gap: var(--space-lg);
}

.catalog-header-panel {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.catalog-header-badges {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.catalog-header-unspsc {
  padding: 2px 8px;
  border-radius: var(--radius);
  background-color: var(--surface-container);
  font-size: var(--text-label-sm);
  font-weight: 600;
  color: var(--muted);
  text-transform: uppercase;
}

.catalog-header-count {
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.catalog-header-title {
  font-size: var(--text-headline-lg);
  font-weight: 700;
  color: var(--on-surface);
  letter-spacing: var(--tracking-headline-lg);
  margin: 0;
}

.catalog-header-desc {
  margin: 0;
  font-size: var(--text-body-md);
  color: var(--muted);
  max-width: 768px;
}

.catalog-header-actions {
  display: flex;
  gap: var(--space-sm);
  margin-top: var(--space-xs);
}

/* ── SLA Panel ── */
.catalog-sla-panel {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.catalog-sla-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.catalog-sla-label {
  font-size: var(--text-label-sm);
  font-weight: 600;
  text-transform: uppercase;
  color: var(--muted);
}

.catalog-sla-icon {
  font-size: 20px;
  color: var(--secondary);
}

.catalog-sla-value {
  font-size: var(--text-tabular-lg);
  font-weight: 700;
  color: var(--on-surface);
  font-variant-numeric: tabular-nums;
}

.catalog-sla-sub {
  font-size: var(--text-label-sm);
  color: var(--secondary);
  font-weight: 500;
}

.catalog-sla-bar {
  width: 100%;
  height: 6px;
  background-color: var(--surface-container-high);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.catalog-sla-fill {
  height: 100%;
  background-color: var(--secondary);
  border-radius: var(--radius-full);
}

.catalog-sla-note {
  font-size: var(--text-body-sm);
  color: var(--muted);
}

/* ── Content Grid ── */
.catalog-content-grid {
  display: grid;
  grid-template-columns: 300px 1fr;
  gap: var(--space-lg);
  align-items: start;
}

.catalog-main {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.catalog-loading,
.catalog-error {
  padding: var(--space-2xl);
  text-align: center;
  font-size: var(--text-body-md);
  color: var(--muted);
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.catalog-error {
  color: var(--error);
  background-color: var(--error-bg);
}

@media (max-width: 1024px) {
  .catalog-header-grid {
    grid-template-columns: 1fr;
  }

  .catalog-content-grid {
    grid-template-columns: 1fr;
  }
}
</style>
