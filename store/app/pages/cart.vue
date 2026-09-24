<script setup lang="ts">
definePageMeta({ title: 'Cart' })

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
  status: string
}

interface QuoteResult {
  quote_id: string
  total_minor: number
  currency: string
  lines: Array<{ line_code: string; unit_price_minor: number; line_total_minor: number }>
  valid_until: string
}

const cart = ref<CartResponse | null>(null)
const loading = ref(true)
const error = ref('')
const updating = ref<string | null>(null)
const quoting = ref(false)
const quote = ref<QuoteResult | null>(null)

async function loadCart() {
  loading.value = true
  error.value = ''
  quote.value = null
  try {
    cart.value = await $fetch<CartResponse>('/api/cart')
  } catch (e: any) {
    if (e?.response?.status === 401) {
      error.value = 'Please sign in to view your cart.'
    } else {
      error.value = 'Failed to load cart.'
    }
  } finally {
    loading.value = false
  }
}

async function updateQuantity(code: string, qty: number) {
  if (qty < 1) return
  updating.value = code
  try {
    await $fetch(`/api/cart/items/${code}`, {
      method: 'PATCH',
      body: { quantity: qty },
    })
    await loadCart()
  } catch {
  } finally {
    updating.value = null
  }
}

async function removeItem(code: string) {
  updating.value = code
  try {
    await $fetch(`/api/cart/items/${code}`, { method: 'DELETE' })
    await loadCart()
  } catch {
  } finally {
    updating.value = null
  }
}

async function requestQuote() {
  quoting.value = true
  try {
    quote.value = await $fetch<QuoteResult>('/api/cart/quote', { method: 'POST' })
  } catch {
  } finally {
    quoting.value = false
  }
}

const groupedBySupplier = computed(() => {
  if (!cart.value?.lines) return []
  const groups: Record<string, CartLine[]> = {}
  for (const line of cart.value.lines) {
    const key = line.supplier_name || 'Unknown Supplier'
    if (!groups[key]) groups[key] = []
    groups[key].push(line)
  }
  return Object.entries(groups).map(([supplier, lines]) => ({ supplier, lines }))
})

const formatPrice = (amount: number, currency: string) => {
  if (currency === 'VND') return `${amount.toLocaleString()}đ`
  return `$${(amount / 100).toFixed(2)}`
}

onMounted(loadCart)
</script>

<template>
  <div class="cart-page">
    <nav class="cart-breadcrumb">
      <a href="/" class="cart-breadcrumb-link">Home</a>
      <span class="cart-breadcrumb-sep">/</span>
      <span class="cart-breadcrumb-current">Cart</span>
    </nav>

    <h1 class="cart-title">Requisition Cart</h1>

    <div v-if="loading" class="cart-loading">Loading cart...</div>
    <div v-else-if="error" class="cart-error">{{ error }}</div>
    <div v-else-if="!cart || cart.lines.length === 0" class="cart-empty">
      <span class="material-symbols-outlined cart-empty-icon">shopping_cart</span>
      <h2 class="cart-empty-title">Your cart is empty</h2>
      <p class="cart-empty-desc">Browse the catalog to add products to your requisition.</p>
      <a href="/catalog" class="cart-empty-cta">Browse Catalog →</a>
    </div>
    <template v-else>
      <div class="cart-layout">
        <div class="cart-lines">
          <div v-for="group in groupedBySupplier" :key="group.supplier" class="cart-supplier-group">
            <div class="cart-supplier-header">
              <span class="material-symbols-outlined">local_shipping</span>
              <span class="cart-supplier-name">{{ group.supplier }}</span>
              <span class="cart-supplier-count">{{ group.lines.length }} item(s)</span>
            </div>
            <div v-for="line in group.lines" :key="line.code" class="cart-line">
              <div class="cart-line-info">
                <span class="cart-line-name">{{ line.product_name }}</span>
                <span class="cart-line-sku">SKU: {{ line.product_code }}</span>
                <span class="cart-line-unit-price">
                  {{ formatPrice(line.unit_price, line.currency) }} / unit
                </span>
              </div>
              <div class="cart-line-qty">
                <button
                  class="cart-qty-btn"
                  :disabled="updating === line.code || line.quantity <= 1"
                  @click="updateQuantity(line.code, line.quantity - 1)"
                >
                  <span class="material-symbols-outlined">remove</span>
                </button>
                <span class="cart-qty-value">{{ line.quantity }}</span>
                <button
                  class="cart-qty-btn"
                  :disabled="updating === line.code"
                  @click="updateQuantity(line.code, line.quantity + 1)"
                >
                  <span class="material-symbols-outlined">add</span>
                </button>
              </div>
              <div class="cart-line-price">
                {{ formatPrice(line.line_total, line.currency) }}
              </div>
              <button
                class="cart-line-remove"
                :disabled="updating === line.code"
                @click="removeItem(line.code)"
              >
                <span class="material-symbols-outlined">close</span>
              </button>
            </div>
          </div>
        </div>

        <div class="cart-summary">
          <h2 class="cart-summary-title">Order Summary</h2>
          <div class="cart-summary-row">
            <span>Items</span>
            <span>{{ cart.lines.length }}</span>
          </div>
          <div class="cart-summary-row cart-summary-total">
            <span>Total</span>
            <span>{{ formatPrice(cart.subtotal, cart.currency) }}</span>
          </div>

          <div v-if="quote" class="cart-quote-result">
            <div class="cart-quote-header">
              <span class="material-symbols-outlined">request_quote</span>
              <span>Volume Quote</span>
            </div>
            <div class="cart-quote-row">
              <span>Quoted Total</span>
              <span>{{ formatPrice(quote.total_minor, quote.currency) }}</span>
            </div>
            <div class="cart-quote-row cart-quote-valid">
              Valid until {{ new Date(quote.valid_until).toLocaleDateString() }}
            </div>
          </div>

          <a href="/checkout" class="cart-checkout-cta">Proceed to Checkout →</a>
          <button
            class="cart-quote-btn"
            :disabled="quoting"
            @click="requestQuote"
          >
            <span class="material-symbols-outlined">request_quote</span>
            {{ quoting ? 'Requesting...' : 'Request Volume Quote' }}
          </button>
          <a href="/catalog" class="cart-continue">Continue Shopping</a>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.cart-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

.cart-breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.cart-breadcrumb-link { color: var(--muted); text-decoration: none; }
.cart-breadcrumb-link:hover { color: var(--secondary); }
.cart-breadcrumb-sep { color: var(--outline-variant); }
.cart-breadcrumb-current { font-weight: 600; color: var(--on-surface); }

.cart-title {
  font-size: var(--text-headline-lg);
  font-weight: 700;
  color: var(--on-surface);
  margin: 0;
}

.cart-loading, .cart-error {
  padding: var(--space-2xl);
  text-align: center;
  font-size: var(--text-body-md);
  color: var(--muted);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.cart-error { color: var(--error); }

.cart-empty {
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

.cart-empty-icon { font-size: 48px; color: var(--outline-variant); }
.cart-empty-title { margin: 0; font-size: var(--text-headline-md); color: var(--on-surface); }
.cart-empty-desc { margin: 0; color: var(--muted); }
.cart-empty-cta {
  padding: var(--space-sm) var(--space-lg);
  background: var(--primary);
  color: var(--on-primary);
  border-radius: var(--radius-lg);
  text-decoration: none;
  font-weight: 600;
  font-size: var(--text-label-md);
}

.cart-layout {
  display: grid;
  grid-template-columns: 1fr 360px;
  gap: var(--space-xl);
  align-items: start;
}

.cart-lines { display: flex; flex-direction: column; gap: var(--space-lg); }

.cart-supplier-group {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.cart-supplier-header {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-md);
  background: var(--surface-container-low);
  font-size: var(--text-label-md);
  font-weight: 600;
  color: var(--on-surface);
}

.cart-supplier-count { margin-left: auto; font-weight: 400; color: var(--muted); font-size: var(--text-label-sm); }

.cart-line {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  padding: var(--space-md);
  border-top: 1px solid var(--border);
}

.cart-line-info { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.cart-line-name { font-weight: 600; color: var(--on-surface); }
.cart-line-sku { font-size: var(--text-label-sm); color: var(--muted); font-family: monospace; }
.cart-line-unit-price { font-size: var(--text-label-sm); color: var(--muted); }

.cart-line-qty {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
}

.cart-qty-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  color: var(--on-surface);
  cursor: pointer;
}

.cart-qty-btn:hover:not(:disabled) { background: var(--surface-container-low); }
.cart-qty-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.cart-qty-btn .material-symbols-outlined { font-size: 16px; }

.cart-qty-value {
  min-width: 32px;
  text-align: center;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.cart-line-price { font-weight: 700; font-variant-numeric: tabular-nums; min-width: 100px; text-align: right; }

.cart-line-remove {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  border-radius: var(--radius);
}

.cart-line-remove:hover { background: var(--error-bg, #fef2f2); color: var(--error); }
.cart-line-remove .material-symbols-outlined { font-size: 16px; }

.cart-summary {
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

.cart-summary-title { margin: 0; font-size: var(--text-headline-sm); font-weight: 700; }

.cart-summary-row {
  display: flex;
  justify-content: space-between;
  font-size: var(--text-body-md);
  color: var(--muted);
}

.cart-summary-total {
  padding-top: var(--space-md);
  border-top: 1px solid var(--border);
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
}

.cart-checkout-cta {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 44px;
  background: var(--primary);
  color: var(--on-primary);
  border-radius: var(--radius-lg);
  text-decoration: none;
  font-weight: 600;
  font-size: var(--text-label-md);
}

.cart-quote-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-xs);
  height: 40px;
  border: 1px solid var(--secondary);
  border-radius: var(--radius-lg);
  background: transparent;
  color: var(--secondary);
  font-family: var(--font-family);
  font-size: var(--text-label-md);
  font-weight: 600;
  cursor: pointer;
}

.cart-quote-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.cart-quote-btn .material-symbols-outlined { font-size: 18px; }

.cart-quote-result {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
  padding: var(--space-md);
  border: 1px solid var(--secondary);
  border-radius: var(--radius-lg);
  background: var(--secondary-fixed);
}

.cart-quote-header {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-weight: 600;
  color: var(--on-surface);
}

.cart-quote-header .material-symbols-outlined { font-size: 18px; color: var(--secondary); }

.cart-quote-row {
  display: flex;
  justify-content: space-between;
  font-size: var(--text-body-sm);
  color: var(--muted);
}

.cart-quote-valid {
  font-size: var(--text-label-sm);
  font-style: italic;
}

.cart-continue {
  text-align: center;
  color: var(--secondary);
  font-size: var(--text-body-sm);
  text-decoration: none;
}

@media (max-width: 768px) {
  .cart-layout { grid-template-columns: 1fr; }
  .cart-line { flex-wrap: wrap; }
}
</style>
