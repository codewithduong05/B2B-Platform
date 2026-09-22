<template>
  <div class="space-y-6">
    <header>
      <h1 class="text-2xl font-bold tracking-tight">Customers</h1>
      <p class="text-sm text-muted mt-1">Manage buyer accounts &amp; verification</p>
    </header>

    <div v-if="status === 'pending'" class="text-muted py-8 text-center">Loading...</div>
    <div v-else-if="error" class="error-banner">Failed to load users. <button class="link" @click="refresh()">Retry</button></div>
    <div v-else-if="!users.length" class="empty-state">No users found.</div>
    <table v-else class="data-table">
      <thead>
        <tr>
          <th>ID</th>
          <th>Email</th>
          <th>Name</th>
          <th>Role</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="u in users" :key="u.id">
          <td class="font-mono text-xs">{{ u.id }}</td>
          <td>{{ u.email }}</td>
          <td>{{ u.full_name ?? '—' }}</td>
          <td>{{ u.role ?? '—' }}</td>
          <td>
            <span class="badge" :class="u.status === 'active' ? 'badge--ok' : 'badge--muted'">
              {{ u.status }}
            </span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
interface User {
  id: number
  email: string
  full_name?: string
  role?: string
  status: string
}

const { data, refresh, status, error } = await useFetch<{ items: User[]; total: number }>('/api/users')

const users = computed(() => data.value?.items ?? [])
</script>

<style scoped>
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

.badge--ok { background: #dcfce7; color: #166534; }
.badge--muted { background: var(--surface-container-high); color: var(--muted); }

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
