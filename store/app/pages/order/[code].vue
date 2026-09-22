<script setup lang="ts">
definePageMeta({ title: 'Order Detail' })

interface OrderLine {
  code: string
  product_code: string
  product_name: string
  quantity: number
  unit_price: number
  line_total: number
}

interface OrderDetail {
  code: string
  status: string
  total: number
  currency: string
  created_at: string
  po_reference: string
  lines: OrderLine[]
  notes: string
}

const route = useRoute()
const code = route.params.code as string
const order = ref<OrderDetail | null>(null)
const loading = ref(true)
const error = ref('')

async function loadOrder() {
  loading.value = true
  error.value = ''
  try {
    order.value = await $fetch<OrderDetail>(`/api/orders/${code}`)
  } catch (e: any) {
    error.value = e?.data?.message || 'Order not found'
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatPrice(amount: number, currency: string) {
  if (currency === 'VND') return `${amount.toLocaleString()}đ`
  return `$${(amount / 100).toFixed(2)}`
}

onMounted(loadOrder)
</script>

<template>
  <div class="order-detail-page">
    <nav class="order-breadcrumb">
      <a href="/" class="order-breadcrumb-link">Home</a>
      <span class="order-breadcrumb-sep">/</span>
      <a href="/orders" class="order-breadcrumb-link">Orders</a>
      <span class="order-breadcrumb-sep">/</span>
      <span class="order-breadcrumb-current">{{ code }}</span>
    </nav>

    <div v-if="loading" class="order-loading">Loading order...</div>
    <div v-else-if="error" class="order-error">{{ error }}</div>
    <template v-else-if="order">
      <div class="order-header">
        <div>
          <h1 class="order-code">{{ order.code }}</h1>
          <span class="order-date">Placed {{ formatDate(order.created_at) }}</span>
        </div>
        <span class="order-status" :class="`order-status--${order.status}`">{{ order.status }}</span>
      </div>

      <div v-if="order.po_reference" class="order-po">
        <span class="order-po-label">PO Reference:</span>
        <span class="order-po-value">{{ order.po_reference }}</span>
      </div>

      <div class="order-lines-card">
        <h2 class="order-section-title">Order Items</h2>
        <div v-for="line in order.lines" :key="line.code" class="order-line">
          <div class="order-line-info">
            <span class="order-line-name">{{ line.product_name }}</span>
            <span class="order-line-sku">SKU: {{ line.product_code }}</span>
          </div>
          <span class="order-line-qty">× {{ line.quantity }}</span>
          <span class="order-line-price">{{ formatPrice(line.line_total, order.currency) }}</span>
        </div>
        <div class="order-total-row">
          <span>Total</span>
          <span class="order-total-value">{{ formatPrice(order.total, order.currency) }}</span>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.order-detail-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
  max-width: 800px;
}

.order-breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.order-breadcrumb-link { color: var(--muted); text-decoration: none; }
.order-breadcrumb-link:hover { color: var(--secondary); }
.order-breadcrumb-sep { color: var(--outline-variant); }
.order-breadcrumb-current { font-weight: 600; color: var(--on-surface); }

.order-loading, .order-error {
  padding: var(--space-2xl);
  text-align: center;
  color: var(--muted);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.order-error { color: var(--error); }

.order-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.order-code { margin: 0; font-size: var(--text-headline-lg); font-weight: 700; color: var(--on-surface); }
.order-date { font-size: var(--text-body-sm); color: var(--muted); }

.order-status {
  padding: 4px 12px;
  border-radius: var(--radius);
  font-size: var(--text-label-sm);
  font-weight: 600;
  text-transform: capitalize;
}

.order-status--pending { background: var(--surface-container); color: var(--muted); }
.order-status--confirmed { background: var(--secondary-fixed); color: var(--on-secondary-fixed); }
.order-status--shipped { background: #dbeafe; color: #1d4ed8; }
.order-status--delivered { background: #dcfce7; color: #16a34a; }

.order-po {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-md);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.order-po-label { font-size: var(--text-label-sm); color: var(--muted); }
.order-po-value { font-weight: 600; font-family: monospace; }

.order-lines-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.order-section-title { margin: 0; font-size: var(--text-headline-sm); font-weight: 700; }

.order-line {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  padding: var(--space-md) 0;
  border-bottom: 1px solid var(--border);
}

.order-line:last-of-type { border-bottom: none; }
.order-line-info { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.order-line-name { font-weight: 600; }
.order-line-sku { font-size: var(--text-label-sm); color: var(--muted); font-family: monospace; }
.order-line-qty { color: var(--muted); }
.order-line-price { font-weight: 700; font-variant-numeric: tabular-nums; }

.order-total-row {
  display: flex;
  justify-content: space-between;
  padding-top: var(--space-md);
  border-top: 2px solid var(--border);
  font-size: var(--text-headline-sm);
  font-weight: 700;
}
</style>
