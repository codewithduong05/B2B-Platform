<template>
  <div class="space-y-6">
    <nav class="breadcrumb">
      <NuxtLink to="/orders" class="breadcrumb-link">Orders</NuxtLink>
      <span class="breadcrumb-sep">/</span>
      <span class="breadcrumb-current">{{ id }}</span>
    </nav>

    <div v-if="loading" class="card">Loading order...</div>
    <div v-else-if="error" class="card error-text">{{ error }}</div>
    <template v-else-if="order">
      <div class="flex items-start justify-between">
        <div>
          <h1 class="text-2xl font-bold">{{ order.code }}</h1>
          <p class="text-sm text-muted mt-1">
            Buyer: {{ order.buyer_name || '—' }}
          </p>
        </div>
        <span class="badge" :class="`badge--${order.status}`">{{ order.status }}</span>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div class="card lg:col-span-2">
          <h2 class="card-title">Order Items</h2>
          <div v-if="order.lines && order.lines.length" class="order-lines">
            <div v-for="line in order.lines" :key="line.code" class="order-line">
              <div class="order-line-info">
                <span class="order-line-name">{{ line.product_name }}</span>
                <span class="order-line-sku">SKU: {{ line.product_code }}</span>
              </div>
              <span class="order-line-qty">× {{ line.quantity }}</span>
              <span class="order-line-price">{{ formatPrice(line.line_total, order.currency) }}</span>
            </div>
          </div>
          <div v-else class="text-muted text-sm">No line items</div>

          <div v-if="order.total" class="order-total-row">
            <span>Total</span>
            <span class="font-bold">{{ formatPrice(order.total, order.currency) }}</span>
          </div>
        </div>

        <div class="card">
          <h2 class="card-title">Info</h2>
          <div class="detail-grid">
            <div class="detail-row">
              <span class="detail-label">Status</span>
              <span class="badge badge--sm" :class="`badge--${order.status}`">{{ order.status }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">PO Reference</span>
              <span class="mono">{{ order.po_reference || '—' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Items</span>
              <span>{{ order.item_count || order.lines?.length || 0 }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Created</span>
              <span>{{ order.created_at ? new Date(order.created_at).toLocaleDateString() : '—' }}</span>
            </div>
            <div v-if="order.delivery_notes" class="detail-row detail-row--full">
              <span class="detail-label">Delivery Notes</span>
              <p class="text-sm mt-1">{{ order.delivery_notes }}</p>
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

const order = ref<any>(null)
const loading = ref(true)
const error = ref('')

function formatPrice(amount: number, currency: string) {
  if (currency === 'VND') return `${amount.toLocaleString()}đ`
  return `$${(amount / 100).toFixed(2)}`
}

async function loadOrder() {
  loading.value = true
  try {
    order.value = await $fetch(`/api/orders/${id}`)
  } catch (e: any) {
    error.value = e?.data?.message || 'Order not found'
  } finally {
    loading.value = false
  }
}

onMounted(loadOrder)
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

.order-lines {
  display: flex;
  flex-direction: column;
}

.order-line {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 0;
  border-bottom: 1px solid var(--border);
}

.order-line:last-of-type { border-bottom: none; }
.order-line-info { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.order-line-name { font-weight: 600; font-size: 0.875rem; }
.order-line-sku { font-size: 0.75rem; color: var(--muted); font-family: monospace; }
.order-line-qty { color: var(--muted); font-size: 0.875rem; }
.order-line-price { font-weight: 700; font-variant-numeric: tabular-nums; }

.order-total-row {
  display: flex;
  justify-content: space-between;
  padding-top: 0.75rem;
  border-top: 2px solid var(--border);
  font-size: 1rem;
  margin-top: 0.5rem;
}

.detail-grid {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.875rem;
}

.detail-row--full {
  flex-direction: column;
  align-items: flex-start;
}

.detail-label {
  color: var(--muted);
  font-weight: 500;
}

.mono { font-family: monospace; }
.error-text { color: var(--error); }
.text-muted { color: var(--muted); }

.badge {
  padding: 4px 12px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: capitalize;
}

.badge--sm { font-size: 0.7rem; padding: 2px 8px; }
.badge--pending { background: var(--surface-container-low); color: var(--muted); }
.badge--confirmed { background: #dbeafe; color: #1d4ed8; }
.badge--shipped { background: #fef3c7; color: #d97706; }
.badge--delivered { background: #dcfce7; color: #16a34a; }
.badge--cancelled { background: #fee2e2; color: #dc2626; }
</style>
