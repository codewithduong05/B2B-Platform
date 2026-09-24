<template>
  <div class="space-y-6">
    <header>
      <h1 class="text-2xl font-bold tracking-tight">Orders</h1>
      <p class="text-sm text-muted mt-1">Monitor purchase orders &amp; dispatch audit</p>
    </header>

    <div class="flex gap-3 items-center">
      <select v-model="statusFilter" class="filter-select">
        <option value="">All statuses</option>
        <option value="pending">Pending Confirmation</option>
        <option value="confirmed">Confirmed</option>
        <option value="shipped">Shipped</option>
        <option value="delivered">Delivered</option>
        <option value="cancelled">Cancelled</option>
      </select>
      <button class="btn btn--ghost" @click="refresh()">Refresh</button>
    </div>

    <div v-if="status === 'pending'" class="text-muted py-8 text-center">Loading...</div>
    <div v-else-if="error" class="error-banner">Failed to load orders. <button class="link" @click="refresh()">Retry</button></div>
    <div v-else-if="!orders.length" class="empty-state">No orders found.</div>
    <table v-else class="data-table">
      <thead>
        <tr>
          <th>Order Code</th>
          <th>Buyer</th>
          <th>Status</th>
          <th>Items</th>
          <th>Total</th>
          <th>Created</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="o in orders" :key="o.id" class="clickable-row" @click="navigateTo(`/orders/${o.id}`)">
          <td class="font-mono text-xs">{{ o.code }}</td>
          <td>{{ o.buyer_name ?? '—' }}</td>
          <td>
            <span class="badge" :class="statusClass(o.status)">{{ o.status }}</span>
          </td>
          <td>{{ o.item_count ?? '—' }}</td>
          <td class="font-mono">{{ o.total_amount != null ? `$${(o.total_amount / 100).toFixed(2)}` : '—' }}</td>
          <td class="text-muted text-xs">{{ o.created_at ? new Date(o.created_at).toLocaleDateString() : '—' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
const statusFilter = ref('')

interface Order {
  id: number
  code: string
  buyer_name?: string
  status: string
  item_count?: number
  total_amount?: number
  created_at?: string
}

const { data, refresh, status, error } = await useFetch<{ items: Order[]; total: number }>('/api/orders', {
  query: computed(() => ({
    ...(statusFilter.value ? { status: statusFilter.value } : {}),
  })),
})

const orders = computed(() => data.value?.items ?? [])

function statusClass(s: string): string {
  if (s === 'pending') return 'badge--warn'
  if (s === 'confirmed' || s === 'shipped') return 'badge--ok'
  if (s === 'cancelled') return 'badge--error'
  return 'badge--muted'
}
</script>

<style scoped>
.filter-select {
  padding: 0.5rem 2rem 0.5rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  font-size: 0.875rem;
  outline: none;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' fill='%2364748b'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 0.5rem center;
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

.clickable-row {
  cursor: pointer;
}

.badge {
  display: inline-block;
  padding: 0.125rem 0.5rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
}

.badge--ok { background: #dcfce7; color: #166534; }
.badge--warn { background: #fef3c7; color: #92400e; }
.badge--error { background: #fef2f2; color: #991b1b; }
.badge--muted { background: var(--surface-container-high); color: var(--muted); }

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

.btn--ghost {
  background: var(--surface-container-low);
  color: var(--text);
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
</style>
