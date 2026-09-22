<script setup lang="ts">
definePageMeta({ title: 'Checkout' })

const loading = ref(false)
const submitting = ref(false)
const error = ref('')
const success = ref(false)
const orderCode = ref('')

const form = ref({
  po_reference: '',
  delivery_notes: '',
})

async function handleCheckout() {
  submitting.value = true
  error.value = ''
  try {
    const result = await $fetch<{ code: string }>('/api/checkout', {
      method: 'POST',
      body: form.value,
    })
    orderCode.value = result.code
    success.value = true
  } catch (e: any) {
    const detail = e?.data?.message || e?.message || 'Checkout failed'
    error.value = detail
  } finally {
    submitting.value = false
  }
}

if (success.value) {
  await navigateTo(`/order/${orderCode.value}`)
}
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

    <div v-if="success" class="checkout-success">
      <span class="material-symbols-outlined checkout-success-icon">check_circle</span>
      <h2>Order Submitted</h2>
      <p>Your order <strong>{{ orderCode }}</strong> has been placed successfully.</p>
      <a :href="`/order/${orderCode}`" class="checkout-success-cta">View Order →</a>
    </div>

    <form v-else class="checkout-form" @submit.prevent="handleCheckout">
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

      <button
        type="submit"
        class="checkout-submit"
        :disabled="submitting"
      >
        <span v-if="submitting" class="material-symbols-outlined checkout-spinner">progress_activity</span>
        {{ submitting ? 'Processing...' : 'Place Order' }}
      </button>
    </form>
  </div>
</template>

<style scoped>
.checkout-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
  max-width: 640px;
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

.checkout-success {
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

.checkout-success-icon { font-size: 48px; color: var(--secondary); }
.checkout-success h2 { margin: 0; color: var(--on-surface); }
.checkout-success p { margin: 0; color: var(--muted); }
.checkout-success-cta {
  padding: var(--space-sm) var(--space-lg);
  background: var(--primary);
  color: var(--on-primary);
  border-radius: var(--radius-lg);
  text-decoration: none;
  font-weight: 600;
}

.checkout-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
}

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
</style>
