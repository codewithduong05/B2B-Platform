<template>
  <div class="space-y-6">
    <header>
      <h1 class="text-2xl font-bold tracking-tight">Inventory</h1>
      <p class="text-sm text-muted mt-1">Stock alerts, low-stock &amp; expiring items</p>
    </header>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div class="card">
        <h2 class="card-title">Low Stock</h2>
        <div v-if="lowStockStatus === 'pending'" class="text-muted py-4 text-center">Loading...</div>
        <div v-else-if="lowStockError" class="error-banner">Failed to load.</div>
        <div v-else-if="!lowStockItems.length" class="empty-state">No low stock items.</div>
        <table v-else class="data-table">
          <thead>
            <tr>
              <th>Product</th>
              <th>Current Stock</th>
              <th>Safety Stock</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(item, i) in lowStockItems" :key="i">
              <td>{{ item.product_name ?? item.sku ?? '—' }}</td>
              <td class="font-mono text-error font-bold">{{ item.current_stock ?? item.quantity ?? '—' }}</td>
              <td class="font-mono">{{ item.safety_stock ?? '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="card">
        <h2 class="card-title">Expiring Soon</h2>
        <div v-if="expiringStatus === 'pending'" class="text-muted py-4 text-center">Loading...</div>
        <div v-else-if="expiringError" class="error-banner">Failed to load.</div>
        <div v-else-if="!expiringItems.length" class="empty-state">No expiring items.</div>
        <table v-else class="data-table">
          <thead>
            <tr>
              <th>Lot</th>
              <th>Product</th>
              <th>Expires</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(item, i) in expiringItems" :key="i">
              <td class="font-mono text-xs">{{ item.lot_code ?? item.code ?? '—' }}</td>
              <td>{{ item.product_name ?? '—' }}</td>
              <td class="text-xs">{{ item.expires_at ?? item.expiry_date ?? '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const { data: lowStockData, status: lowStockStatus, error: lowStockError } = await useFetch<{ items: unknown[] }>('/api/inventory/low-stock')
const { data: expiringData, status: expiringStatus, error: expiringError } = await useFetch<{ items: unknown[] }>('/api/inventory/expiring')

const lowStockItems = computed(() => {
  const d = lowStockData.value
  if (Array.isArray(d)) return d
  if (d && typeof d === 'object' && 'items' in d && Array.isArray((d as { items: unknown[] }).items)) {
    return (d as { items: unknown[] }).items
  }
  return []
})

const expiringItems = computed(() => {
  const d = expiringData.value
  if (Array.isArray(d)) return d
  if (d && typeof d === 'object' && 'items' in d && Array.isArray((d as { items: unknown[] }).items)) {
    return (d as { items: unknown[] }).items
  }
  return []
})
</script>

<style scoped>
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

.data-table {
  width: 100%;
  border-collapse: collapse;
}

.data-table th,
.data-table td {
  padding: 0.5rem 0.75rem;
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
}

.data-table tbody tr:hover {
  background: var(--surface-container-low);
}

.text-error { color: var(--error); }

.error-banner {
  padding: 0.75rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 8px;
  color: #991b1b;
  font-size: 0.875rem;
}

.empty-state {
  text-align: center;
  padding: 2rem;
  color: var(--muted);
  font-size: 0.875rem;
}
</style>
