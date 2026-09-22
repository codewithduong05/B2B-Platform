<script setup lang="ts">
definePageMeta({ title: 'Quick Order' })

const skuInput = ref('')
const qtyInput = ref(5)
const adding = ref(false)
const addedMessage = ref('')

const { addToCart } = useCart()

async function handleAdd() {
  if (!skuInput.value.trim()) return
  adding.value = true
  addedMessage.value = ''
  try {
    await addToCart(skuInput.value.trim(), qtyInput.value)
    addedMessage.value = `Added ${qtyInput.value}× ${skuInput.value} to cart`
    skuInput.value = ''
    qtyInput.value = 5
  } catch {
    addedMessage.value = 'Failed to add item. Check SKU and try again.'
  } finally {
    adding.value = false
  }
}
</script>

<template>
  <div class="quick-order-page">
    <nav class="qo-breadcrumb">
      <a href="/" class="qo-breadcrumb-link">Home</a>
      <span class="qo-breadcrumb-sep">/</span>
      <span class="qo-breadcrumb-current">Quick Order</span>
    </nav>

    <h1 class="qo-title">Quick Order</h1>
    <p class="qo-desc">Add products directly by SKU. Enter the product code and quantity to add to your requisition cart.</p>

    <form class="qo-form" @submit.prevent="handleAdd">
      <div class="qo-field">
        <label class="qo-label" for="sku">Product SKU</label>
        <input
          id="sku"
          v-model="skuInput"
          type="text"
          class="qo-input"
          placeholder="e.g. SKU-12345"
          required
        />
      </div>
      <div class="qo-field">
        <label class="qo-label" for="qty">Quantity</label>
        <input
          id="qty"
          v-model.number="qtyInput"
          type="number"
          class="qo-input"
          min="1"
          max="9999"
          required
        />
      </div>
      <button type="submit" class="qo-submit" :disabled="adding || !skuInput.trim()">
        <span class="material-symbols-outlined">add_shopping_cart</span>
        {{ adding ? 'Adding...' : 'Add to Cart' }}
      </button>
    </form>

    <div v-if="addedMessage" class="qo-message">{{ addedMessage }}</div>

    <div class="qo-links">
      <a href="/catalog" class="qo-link">Browse Catalog →</a>
      <a href="/cart" class="qo-link">View Cart →</a>
    </div>
  </div>
</template>

<style scoped>
.quick-order-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
  max-width: 560px;
}

.qo-breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.qo-breadcrumb-link { color: var(--muted); text-decoration: none; }
.qo-breadcrumb-link:hover { color: var(--secondary); }
.qo-breadcrumb-sep { color: var(--outline-variant); }
.qo-breadcrumb-current { font-weight: 600; color: var(--on-surface); }

.qo-title { margin: 0; font-size: var(--text-headline-lg); font-weight: 700; }
.qo-desc { margin: 0; color: var(--muted); }

.qo-form {
  display: flex;
  gap: var(--space-md);
  align-items: flex-end;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
}

.qo-field { display: flex; flex-direction: column; gap: var(--space-xs); flex: 1; }
.qo-label { font-size: var(--text-label-md); font-weight: 600; }

.qo-input {
  height: 44px;
  padding: 0 var(--space-md);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  font-family: var(--font-family);
  font-size: var(--text-body-md);
  outline: none;
}

.qo-input:focus { border-color: var(--secondary); box-shadow: 0 0 0 2px var(--secondary-fixed); }

.qo-submit {
  height: 44px;
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: 0 var(--space-lg);
  border: none;
  border-radius: var(--radius-lg);
  background: var(--primary);
  color: var(--on-primary);
  font-family: var(--font-family);
  font-size: var(--text-label-md);
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
}

.qo-submit:disabled { opacity: 0.5; cursor: not-allowed; }

.qo-message {
  padding: var(--space-md);
  border-radius: var(--radius-lg);
  background: var(--surface);
  border: 1px solid var(--border);
  font-size: var(--text-body-sm);
  color: var(--on-surface);
}

.qo-links {
  display: flex;
  gap: var(--space-lg);
}

.qo-link { color: var(--secondary); font-weight: 600; text-decoration: none; }
.qo-link:hover { text-decoration: underline; }
</style>
