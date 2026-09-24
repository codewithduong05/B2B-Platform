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

interface ShipmentEvent {
  status: string
  timestamp: string
  description: string
  location?: string
}

interface OrderDetail {
  code: string
  status: string
  total: number
  currency: string
  created_at: string
  updated_at: string
  po_reference: string
  delivery_notes: string
  lines: OrderLine[]
  notes: string
  shipment_events: ShipmentEvent[]
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

const statusSteps = ['pending', 'confirmed', 'shipped', 'delivered']
const statusLabels: Record<string, string> = {
  pending: 'Pending',
  confirmed: 'Confirmed',
  shipped: 'Shipped',
  delivered: 'Delivered',
}

const currentStepIndex = computed(() => {
  if (!order.value) return -1
  return statusSteps.indexOf(order.value.status)
})

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

      <!-- Status Timeline -->
      <div class="order-timeline">
        <div
          v-for="(step, i) in statusSteps"
          :key="step"
          class="timeline-step"
          :class="{
            'timeline-step--completed': i <= currentStepIndex,
            'timeline-step--current': i === currentStepIndex,
          }"
        >
          <div class="timeline-dot">
            <span v-if="i < currentStepIndex" class="material-symbols-outlined">check</span>
            <span v-else-if="i === currentStepIndex" class="material-symbols-outlined">radio_button_checked</span>
            <span v-else class="material-symbols-outlined">radio_button_unchecked</span>
          </div>
          <span class="timeline-label">{{ statusLabels[step] }}</span>
          <div v-if="i < statusSteps.length - 1" class="timeline-connector" :class="{ 'timeline-connector--active': i < currentStepIndex }" />
        </div>
      </div>

      <!-- PO Reference -->
      <div v-if="order.po_reference" class="order-po">
        <span class="material-symbols-outlined">description</span>
        <span class="order-po-label">PO Reference:</span>
        <span class="order-po-value">{{ order.po_reference }}</span>
      </div>

      <!-- Delivery Notes -->
      <div v-if="order.delivery_notes" class="order-notes">
        <span class="material-symbols-outlined">note</span>
        <div>
          <span class="order-notes-label">Delivery Notes</span>
          <p class="order-notes-body">{{ order.delivery_notes }}</p>
        </div>
      </div>

      <div class="order-layout">
        <!-- Order Items -->
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

        <!-- Shipment Events -->
        <div v-if="order.shipment_events && order.shipment_events.length" class="order-shipment-card">
          <h2 class="order-section-title">
            <span class="material-symbols-outlined">local_shipping</span>
            Shipment Tracking
          </h2>
          <div class="shipment-events">
            <div v-for="(event, i) in order.shipment_events" :key="i" class="shipment-event">
              <div class="shipment-event-dot" />
              <div class="shipment-event-content">
                <span class="shipment-event-status">{{ event.status }}</span>
                <span class="shipment-event-desc">{{ event.description }}</span>
                <span class="shipment-event-time">{{ formatDate(event.timestamp) }}</span>
                <span v-if="event.location" class="shipment-event-location">{{ event.location }}</span>
              </div>
            </div>
          </div>
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
  max-width: 960px;
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
.order-status--cancelled { background: #fee2e2; color: #dc2626; }

/* ── Timeline ── */
.order-timeline {
  display: flex;
  align-items: center;
  gap: 0;
  padding: var(--space-lg);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.timeline-step {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  flex: 1;
}

.timeline-dot .material-symbols-outlined {
  font-size: 20px;
  color: var(--outline-variant);
}

.timeline-step--completed .timeline-dot .material-symbols-outlined {
  color: var(--secondary);
}

.timeline-step--current .timeline-dot .material-symbols-outlined {
  color: var(--primary);
}

.timeline-label {
  font-size: var(--text-label-sm);
  font-weight: 600;
  color: var(--muted);
  white-space: nowrap;
}

.timeline-step--completed .timeline-label,
.timeline-step--current .timeline-label {
  color: var(--on-surface);
}

.timeline-connector {
  flex: 1;
  height: 2px;
  background: var(--surface-container-high);
  margin: 0 var(--space-sm);
}

.timeline-connector--active {
  background: var(--secondary);
}

/* ── PO & Notes ── */
.order-po, .order-notes {
  display: flex;
  align-items: flex-start;
  gap: var(--space-sm);
  padding: var(--space-md);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.order-po .material-symbols-outlined,
.order-notes .material-symbols-outlined {
  font-size: 20px;
  color: var(--muted);
  margin-top: 2px;
}

.order-po-label, .order-notes-label { font-size: var(--text-label-sm); color: var(--muted); display: block; }
.order-po-value { font-weight: 600; font-family: monospace; }
.order-notes-body { margin: 4px 0 0; font-size: var(--text-body-sm); color: var(--on-surface); }

/* ── Layout ── */
.order-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-lg);
  align-items: start;
}

.order-lines-card, .order-shipment-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.order-section-title {
  margin: 0;
  font-size: var(--text-headline-sm);
  font-weight: 700;
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.order-section-title .material-symbols-outlined { font-size: 20px; color: var(--secondary); }

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

/* ── Shipment Events ── */
.shipment-events {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.shipment-event {
  display: flex;
  gap: var(--space-md);
}

.shipment-event-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--secondary);
  margin-top: 6px;
  flex-shrink: 0;
}

.shipment-event-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.shipment-event-status {
  font-weight: 600;
  font-size: var(--text-body-sm);
  color: var(--on-surface);
  text-transform: capitalize;
}

.shipment-event-desc {
  font-size: var(--text-body-sm);
  color: var(--muted);
}

.shipment-event-time {
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.shipment-event-location {
  font-size: var(--text-label-sm);
  color: var(--muted);
  font-style: italic;
}

@media (max-width: 768px) {
  .order-layout { grid-template-columns: 1fr; }
  .order-timeline { flex-wrap: wrap; gap: var(--space-sm); }
}
</style>
