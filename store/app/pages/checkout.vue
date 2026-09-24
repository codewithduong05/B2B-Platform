<script setup lang="ts">
definePageMeta({ title: 'Checkout' })

interface CartLine {
  code: string
  product_code: string
  product_name: string
  supplier_name: string
  quantity: number
  unit_price: number
  currency: string
  line_total: number
}

interface CartResponse {
  code: string
  lines: CartLine[]
  subtotal: number
  currency: string
}

const submitting = ref(false)
const error = ref('')
const cart = ref<CartResponse | null>(null)
const loadingCart = ref(true)

const form = ref({
  po_reference: '',
  delivery_notes: '',
})

async function loadCart() {
  try {
    cart.value = await $fetch<CartResponse>('/api/cart')
  } catch (e: any) {
    if (e?.response?.status === 401) {
      error.value = 'Please sign in to proceed with checkout.'
    } else {
      error.value = 'Failed to load cart.'
    }
  } finally {
    loadingCart.value = false
  }
}

async function handleCheckout() {
  submitting.value = true
  error.value = ''
  try {
    const result = await $fetch<{ code: string }>('/api/checkout', {
      method: 'POST',
      body: form.value,
    })
    await navigateTo(`/order/confirm/${result.code}`)
  } catch (e: any) {
    const detail = e?.data?.message || e?.message || 'Checkout failed'
    error.value = detail
  } finally {
    submitting.value = false
  }
}

const formatPrice = (amount: number, currency: string) => {
  if (currency === 'VND') return `${amount.toLocaleString()}đ`
  return `$${(amount / 100).toFixed(2)}`
}

onMounted(loadCart)
</script>

<template>
  <div class="checkout-page">
    <nav class="checkout-breadcrumb">
      <a href="/" class="checkout-breadcrumb-link">Home</a>
      <span class="checkout-breadcrumb-sep">/</span>
      <a href="/cart" class="checkout-breadcrumb-link">Cart</a>
      <span class="checkout-breadcrumb-sep">/</span>
      <span class="checkout-breadcrumb-current">Checkout</span>
    </nav>

    <h1 class="checkout-title">Checkout</h1>

    <div v-if="error" class="checkout-error">
      <span class="material-symbols-outlined">error</span>
      <span>{{ error }}</span>
    </div>

    <div v-if="loadingCart" class="checkout-loading">Loading cart...</div>

    <template v-else-if="cart && cart.lines.length > 0">
      <div class="checkout-layout">
        <!-- Left: Form -->
        <form class="checkout-form" @submit.prevent="handleCheckout">
          <!-- Cart Review -->
          <div class="checkout-section">
            <h2 class="checkout-section-title">
              <span class="material-symbols-outlined">shopping_cart</span>
              Order Review
            </h2>
            <div class="checkout-review-items">
              <div v-for="line in cart.lines" :key="line.code" class="checkout-review-line">
                <div class="checkout-review-info">
                  <span class="checkout-review-name">{{ line.product_name }}</span>
                  <span class="checkout-review-sku">SKU: {{ line.product_code }}</span>
                </div>
                <div class="checkout-review-qty">
                  {{ line.quantity }} × {{ formatPrice(line.unit_price, line.currency) }}
                </div>
                <div class="checkout-review-total">
                  {{ formatPrice(line.line_total, line.currency) }}
                </div>
              </div>
            </div>
          </div>

          <!-- PO Reference -->
          <div class="checkout-section">
            <h2 class="checkout-section-title">
              <span class="material-symbols-outlined">description</span>
              Purchase Order
            </h2>
            <div class="checkout-field">
              <label class="checkout-label" for="po_ref">PO Reference (optional)</label>
              <input
                id="po_ref"
                v-model="form.po_reference"
                type="text"
                class="checkout-input"
                placeholder="e.g. PO-2026-00123"
              />
            </div>
            <div class="checkout-field">
              <label class="checkout-label" for="notes">Delivery Notes (optional)</label>
              <textarea
                id="notes"
                v-model="form.delivery_notes"
                class="checkout-textarea"
                rows="3"
                placeholder="Special delivery instructions..."
              />
            </div>
          </div>

          <button
            type="submit"
            class="checkout-submit"
            :disabled="submitting"
          >
            <span v-if="submitting" class="material-symbols-outlined checkout-spinner">progress_activity</span>
            {{ submitting ? 'Processing...' : 'Place Order' }}
          </button>
        </form>

        <!-- Right: Summary -->
        <div class="checkout-summary">
          <h2 class="checkout-summary-title">Order Summary</h2>
          <div class="checkout-summary-row">
            <span>Items</span>
            <span>{{ cart.lines.length }}</span>
          </div>
          <div class="checkout-summary-row checkout-summary-total">
            <span>Total</span>
            <span>{{ formatPrice(cart.subtotal, cart.currency) }}</span>
          </div>
        </div>
      </div>
    </template>

    <div v-else class="checkout-empty">
      <p>Your cart is empty.</p>
      <a href="/catalog" class="checkout-empty-cta">Browse Catalog →</a>
    </div>
  </div>
</template>

<style scoped>
.checkout-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
  max-width: 960px;
}

.checkout-breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.checkout-breadcrumb-link { color: var(--muted); text-decoration: none; }
.checkout-breadcrumb-link:hover { color: var(--secondary); }
.checkout-breadcrumb-sep { color: var(--outline-variant); }
.checkout-breadcrumb-current { font-weight: 600; color: var(--on-surface); }

.checkout-title {
  font-size: var(--text-headline-lg);
  font-weight: 700;
  color: var(--on-surface);
  margin: 0;
}

.checkout-error {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-md);
  border-radius: var(--radius-lg);
  background-color: var(--error-bg, #fef2f2);
  color: var(--error);
  font-size: var(--text-body-sm);
}

.checkout-loading {
  padding: var(--space-2xl);
  text-align: center;
  color: var(--muted);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.checkout-layout {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: var(--space-xl);
  align-items: start;
}

.checkout-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

.checkout-section {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.checkout-section-title {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  margin: 0;
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
}

.checkout-section-title .material-symbols-outlined {
  font-size: 20px;
  color: var(--secondary);
}

.checkout-review-items {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.checkout-review-line {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  padding: var(--space-sm) 0;
  border-bottom: 1px solid var(--surface-container-high);
}

.checkout-review-line:last-child { border-bottom: none; }

.checkout-review-info { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.checkout-review-name { font-weight: 600; color: var(--on-surface); font-size: var(--text-body-sm); }
.checkout-review-sku { font-size: var(--text-label-sm); color: var(--muted); font-family: monospace; }
.checkout-review-qty { font-size: var(--text-body-sm); color: var(--muted); min-width: 120px; }
.checkout-review-total { font-weight: 600; color: var(--on-surface); min-width: 80px; text-align: right; }

.checkout-field { display: flex; flex-direction: column; gap: var(--space-xs); }
.checkout-label { font-size: var(--text-label-md); font-weight: 600; color: var(--on-surface); }

.checkout-input, .checkout-textarea {
  width: 100%;
  padding: var(--space-sm) var(--space-md);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  font-family: var(--font-family);
  font-size: var(--text-body-md);
  color: var(--on-surface);
  outline: none;
}

.checkout-input:focus, .checkout-textarea:focus {
  border-color: var(--secondary);
  box-shadow: 0 0 0 2px var(--secondary-fixed);
}

.checkout-textarea { resize: vertical; }

.checkout-submit {
  width: 100%;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-sm);
  border: none;
  border-radius: var(--radius-lg);
  background: var(--primary);
  color: var(--on-primary);
  font-family: var(--font-family);
  font-size: var(--text-label-md);
  font-weight: 600;
  cursor: pointer;
}

.checkout-submit:disabled { opacity: 0.5; cursor: not-allowed; }
.checkout-spinner { animation: spin 1s linear infinite; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

.checkout-summary {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  position: sticky;
  top: var(--space-lg);
}

.checkout-summary-title { margin: 0; font-size: var(--text-headline-sm); font-weight: 700; }

.checkout-summary-row {
  display: flex;
  justify-content: space-between;
  font-size: var(--text-body-md);
  color: var(--muted);
}

.checkout-summary-total {
  padding-top: var(--space-md);
  border-top: 1px solid var(--border);
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
}

.checkout-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-md);
  padding: var(--space-2xl);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  text-align: center;
  color: var(--muted);
}

.checkout-empty-cta {
  padding: var(--space-sm) var(--space-lg);
  background: var(--primary);
  color: var(--on-primary);
  border-radius: var(--radius-lg);
  text-decoration: none;
  font-weight: 600;
}

@media (max-width: 768px) {
  .checkout-layout { grid-template-columns: 1fr; }
}
</style>
