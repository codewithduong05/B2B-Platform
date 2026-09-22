<template>
  <div class="space-y-6">
    <header class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold tracking-tight">Products</h1>
        <p class="text-sm text-muted mt-1">Manage product master data &amp; SKU governance</p>
      </div>
      <button class="btn btn--primary">
        <span class="material-symbols-outlined text-[18px]">add_box</span>
        Add Product
      </button>
    </header>

    <div class="flex gap-3 items-center">
      <div class="search-wrapper">
        <span class="material-symbols-outlined search-ico">search</span>
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search products..."
          class="search-field"
          @keyup.enter="refresh()"
        />
      </div>
      <button class="btn btn--ghost" @click="refresh()">Refresh</button>
    </div>

    <div v-if="status === 'pending'" class="text-muted py-8 text-center">Loading...</div>
    <div v-else-if="error" class="error-banner">Failed to load products. <button class="link" @click="refresh()">Retry</button></div>
    <div v-else-if="!products.length" class="empty-state">No products found.</div>
    <table v-else class="data-table">
      <thead>
        <tr>
          <th>SKU</th>
          <th>Name</th>
          <th>Category</th>
          <th>Brand</th>
          <th>Supplier</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="p in products" :key="p.id">
          <td class="font-mono text-xs">{{ p.code }}</td>
          <td>{{ p.name }}</td>
          <td>{{ p.category_name ?? '—' }}</td>
          <td>{{ p.brand_name ?? '—' }}</td>
          <td>{{ p.supplier_name ?? '—' }}</td>
          <td>
            <span class="badge" :class="p.status === 'active' ? 'badge--ok' : 'badge--muted'">
              {{ p.status }}
            </span>
          </td>
        </tr>
      </tbody>
    </table>

    <div v-if="totalCount > pageSize" class="pagination">
      <button :disabled="page <= 1" class="btn btn--ghost" @click="page--; refresh()">Prev</button>
      <span class="text-sm text-muted">Page {{ page }} of {{ Math.ceil(totalCount / pageSize) }}</span>
      <button :disabled="page >= Math.ceil(totalCount / pageSize)" class="btn btn--ghost" @click="page++; refresh()">Next</button>
    </div>
  </div>
</template>

<script setup lang="ts">
const page = ref(1)
const pageSize = 20
const searchQuery = ref('')

interface Product {
  id: number
  code: string
  name: string
  category_name?: string
  brand_name?: string
  supplier_name?: string
  status: string
}

const { data, refresh, status, error } = await useFetch<{ items: Product[]; total: number }>('/api/catalog/products', {
  query: computed(() => ({
    page: page.value,
    page_size: pageSize,
    ...(searchQuery.value ? { q: searchQuery.value } : {}),
  })),
})

const products = computed(() => data.value?.items ?? [])
const totalCount = computed(() => data.value?.total ?? 0)
</script>

<style scoped>
.search-wrapper {
  position: relative;
  flex: 1;
  max-width: 400px;
}

.search-ico {
  position: absolute;
  left: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  font-size: 18px;
  color: var(--muted);
}

.search-field {
  width: 100%;
  padding: 0.5rem 0.75rem 0.5rem 2.5rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  font-size: 0.875rem;
  outline: none;
}

.search-field:focus {
  border-color: var(--secondary);
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
}

.data-table th,
.data-table td {
  padding: 0.75rem 1rem;
  text-align: left;
  border-bottom: 1px solid var(--border);
  font-size: 0.875rem;
}

.data-table th {
  font-weight: 600;
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--muted);
  background: var(--surface-container-low);
}

.data-table tbody tr:hover {
  background: var(--surface-container-low);
}

.badge {
  display: inline-block;
  padding: 0.125rem 0.5rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
}

.badge--ok {
  background: #dcfce7;
  color: #166534;
}

.badge--muted {
  background: var(--surface-container-high);
  color: var(--muted);
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 1rem;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: opacity 0.15s;
}

.btn--primary {
  background: var(--secondary);
  color: var(--on-secondary);
}

.btn--ghost {
  background: var(--surface-container-low);
  color: var(--text);
}

.btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.error-banner {
  padding: 1rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 8px;
  color: #991b1b;
  font-size: 0.875rem;
}

.empty-state {
  text-align: center;
  padding: 3rem;
  color: var(--muted);
}

.link {
  background: none;
  border: none;
  color: var(--secondary);
  text-decoration: underline;
  cursor: pointer;
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
}
</style>
