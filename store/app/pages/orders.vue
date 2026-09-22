<script setup lang="ts">
definePageMeta({ title: 'Orders' })

interface OrderSummary {
  code: string
  status: string
  total: number
  currency: string
  created_at: string
  line_count: number
}

const orders = ref<OrderSummary[]>([])
const loading = ref(true)
const error = ref('')

async function loadOrders() {
  loading.value = true
  error.value = ''
  try {
    const data = await $fetch<{ items: OrderSummary[] }>('/api/orders')
    orders.value = data.items || []
  } catch (e: any) {
    if (e?.response?.status === 401) {
      error.value = 'Please sign in to view your orders.'
    } else {
      error.value = 'Failed to load orders.'
    }
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function formatPrice(amount: number, currency: string) {
  if (currency === 'VND') return `${amount.toLocaleString()}đ`
  return `$${(amount / 100).toFixed(2)}`
}

onMounted(loadOrders)
</script>

<template>
  <div class="orders-page">
    <nav class="orders-breadcrumb">
      <a href="/" class="orders-breadcrumb-link">Home</a>
      <span class="orders-breadcrumb-sep">/</span>
      <span class="orders-breadcrumb-current">Orders</span>
    </nav>

    <h1 class="orders-title">Order History</h1>

    <div v-if="loading" class="orders-loading">Loading orders...</div>
    <div v-else-if="error" class="orders-error">{{ error }}</div>
    <div v-else-if="orders.length === 0" class="orders-empty">
      <span class="material-symbols-outlined orders-empty-icon">receipt_long</span>
      <h2>No orders yet</h2>
      <p>Your procurement history will appear here.</p>
      <a href="/catalog" class="orders-empty-cta">Browse Catalog →</a>
    </div>
    <div v-else class="orders-list">
      <a v-for="order in orders" :key="order.code" :href="`/order/${order.code}`" class="order-card">
        <div class="order-card-left">
          <span class="order-code">{{ order.code }}</span>
          <span class="order-date">{{ formatDate(order.created_at) }}</span>
        </div>
        <div class="order-card-center">
          <span class="order-status" :class="`order-status--${order.status}`">{{ order.status }}</span>
          <span class="order-lines">{{ order.line_count }} item(s)</span>
        </div>
        <div class="order-card-right">
          <span class="order-total">{{ formatPrice(order.total, order.currency) }}</span>
          <span class="material-symbols-outlined order-arrow">chevron_right</span>
        </div>
      </a>
    </div>
  </div>
</template>

<style scoped>
.orders-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

.orders-breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.orders-breadcrumb-link { color: var(--muted); text-decoration: none; }
.orders-breadcrumb-link:hover { color: var(--secondary); }
.orders-breadcrumb-sep { color: var(--outline-variant); }
.orders-breadcrumb-current { font-weight: 600; color: var(--on-surface); }

.orders-title {
  font-size: var(--text-headline-lg);
  font-weight: 700;
  color: var(--on-surface);
  margin: 0;
}

.orders-loading, .orders-error {
  padding: var(--space-2xl);
  text-align: center;
  color: var(--muted);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.orders-error { color: var(--error); }

.orders-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-md);
  padding: var(--space-2xl);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  text-align: center;
}

.orders-empty-icon { font-size: 48px; color: var(--outline-variant); }
.orders-empty h2 { margin: 0; color: var(--on-surface); }
.orders-empty p { margin: 0; color: var(--muted); }
.orders-empty-cta {
  padding: var(--space-sm) var(--space-lg);
  background: var(--primary);
  color: var(--on-primary);
  border-radius: var(--radius-lg);
  text-decoration: none;
  font-weight: 600;
}

.orders-list { display: flex; flex-direction: column; gap: var(--space-sm); }

.order-card {
  display: flex;
  align-items: center;
  gap: var(--space-lg);
  padding: var(--space-md) var(--space-lg);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  text-decoration: none;
  transition: box-shadow 0.15s ease;
}

.order-card:hover { box-shadow: var(--shadow-2); }

.order-card-left { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.order-code { font-weight: 700; color: var(--on-surface); font-family: monospace; }
.order-date { font-size: var(--text-label-sm); color: var(--muted); }

.order-card-center { display: flex; flex-direction: column; align-items: center; gap: 4px; }

.order-status {
  padding: 2px 8px;
  border-radius: var(--radius);
  font-size: var(--text-label-sm);
  font-weight: 600;
  text-transform: capitalize;
}

.order-status--pending { background: var(--surface-container); color: var(--muted); }
.order-status--confirmed { background: var(--secondary-fixed); color: var(--on-secondary-fixed); }
.order-status--shipped { background: #dbeafe; color: #1d4ed8; }
.order-status--delivered { background: #dcfce7; color: #16a34a; }
.order-status--cancelled { background: var(--error-bg, #fef2f2); color: var(--error); }

.order-lines { font-size: var(--text-label-sm); color: var(--muted); }

.order-card-right { display: flex; align-items: center; gap: var(--space-sm); }
.order-total { font-weight: 700; font-variant-numeric: tabular-nums; color: var(--on-surface); }
.order-arrow { color: var(--outline-variant); }
</style>
