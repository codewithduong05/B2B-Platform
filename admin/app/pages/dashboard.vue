<template>
  <div class="space-y-6">
    <header>
      <h1 class="text-2xl font-bold tracking-tight">Dashboard</h1>
      <p class="text-sm text-muted mt-1">Operational overview — real-time data from backend</p>
    </header>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      <div class="stat-card">
        <div class="stat-label">Total Products</div>
        <div class="stat-value">{{ stats?.productCount ?? '—' }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Active Orders</div>
        <div class="stat-value">{{ stats?.orderCount ?? '—' }}</div>
      </div>
      <div class="stat-card stat-card--warning">
        <div class="stat-label">Low Stock Items</div>
        <div class="stat-value">{{ stats?.lowStockItems ?? '—' }}</div>
      </div>
    </div>

    <div class="card">
      <h2 class="card-title">Quick Actions</h2>
      <div class="flex gap-3 flex-wrap">
        <NuxtLink to="/products" class="btn btn--primary">Manage Products</NuxtLink>
        <NuxtLink to="/orders" class="btn btn--secondary">View Orders</NuxtLink>
        <NuxtLink to="/inventory" class="btn btn--secondary">Check Inventory</NuxtLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const { data: stats } = await useFetch('/api/home/stats')
</script>

<style scoped>
.stat-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
}

.stat-card--warning {
  border-color: var(--error);
}

.stat-label {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--muted);
  margin-bottom: 0.25rem;
}

.stat-value {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--text);
}

.stat-card--warning .stat-value {
  color: var(--error);
}

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

.btn {
  display: inline-flex;
  align-items: center;
  padding: 0.5rem 1rem;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
  text-decoration: none;
  transition: opacity 0.15s;
}

.btn--primary {
  background: var(--secondary);
  color: var(--on-secondary);
}

.btn--secondary {
  background: var(--surface-container-low);
  color: var(--text);
}

.btn:hover {
  opacity: 0.9;
}
</style>
