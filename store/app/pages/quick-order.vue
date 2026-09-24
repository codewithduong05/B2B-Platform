<script setup lang="ts">
definePageMeta({ title: 'Quick Order' })

interface OrderLine {
  sku: string
  quantity: number
}

const lines = ref<OrderLine[]>([{ sku: '', quantity: 1 }])
const adding = ref(false)
const messages = ref<{ type: 'success' | 'error'; text: string }[]>([])

const { addToCart } = useCart()

function addLine() {
  lines.value.push({ sku: '', quantity: 1 })
}

function removeLine(index: number) {
  if (lines.value.length > 1) {
    lines.value.splice(index, 1)
  }
}

function handleCsvUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  const reader = new FileReader()
  reader.onload = (e) => {
    const text = e.target?.result as string
    const parsed = text
      .split('\n')
      .map((row) => row.split(',').map((cell) => cell.trim()))
      .filter(([sku, qty]) => sku && qty && !isNaN(Number(qty)))

    if (parsed.length > 0) {
      lines.value = parsed.map(([sku, qty]) => ({
        sku,
        quantity: Number(qty) || 1,
      }))
    }
  }
  reader.readAsText(file)
  input.value = ''
}

async function handleAddAll() {
  adding.value = true
  messages.value = []

  const validLines = lines.value.filter((l) => l.sku.trim())
  if (validLines.length === 0) {
    messages.value.push({ type: 'error', text: 'Please enter at least one SKU.' })
    adding.value = false
    return
  }

  let added = 0
  let failed = 0

  for (const line of validLines) {
    try {
      await addToCart(line.sku.trim(), line.quantity)
      added++
    } catch {
      failed++
    }
  }

  if (added > 0) {
    messages.value.push({ type: 'success', text: `Added ${added} item(s) to cart.` })
  }
  if (failed > 0) {
    messages.value.push({ type: 'error', text: `${failed} item(s) failed. Check SKUs and try again.` })
  }

  adding.value = false
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
    <p class="qo-desc">Add products directly by SKU. Enter multiple lines or upload a CSV file.</p>

    <!-- Multi-line Entry -->
    <div class="qo-lines-card">
      <div v-for="(line, i) in lines" :key="i" class="qo-line">
        <div class="qo-line-fields">
          <div class="qo-field qo-field--sku">
            <label class="qo-label">SKU</label>
            <input
              v-model="line.sku"
              type="text"
              class="qo-input"
              placeholder="e.g. SKU-12345"
            />
          </div>
          <div class="qo-field qo-field--qty">
            <label class="qo-label">Qty</label>
            <input
              v-model.number="line.quantity"
              type="number"
              class="qo-input"
              min="1"
              max="9999"
            />
          </div>
          <button
            class="qo-line-remove"
            :disabled="lines.length <= 1"
            @click="removeLine(i)"
          >
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>
      </div>

      <div class="qo-line-actions">
        <button class="qo-add-line" @click="addLine">
          <span class="material-symbols-outlined">add</span>
          Add Line
        </button>
        <label class="qo-csv-upload">
          <span class="material-symbols-outlined">upload_file</span>
          Upload CSV
          <input type="file" accept=".csv" class="qo-csv-input" @change="handleCsvUpload" />
        </label>
      </div>
    </div>

    <!-- Messages -->
    <div v-if="messages.length" class="qo-messages">
      <div v-for="(msg, i) in messages" :key="i" class="qo-message" :class="`qo-message--${msg.type}`">
        {{ msg.text }}
      </div>
    </div>

    <!-- Actions -->
    <div class="qo-actions">
      <button class="qo-submit" :disabled="adding" @click="handleAddAll">
        <span class="material-symbols-outlined">add_shopping_cart</span>
        {{ adding ? 'Adding...' : 'Add All to Cart' }}
      </button>
    </div>

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
  max-width: 640px;
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

.qo-lines-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.qo-line {
  display: flex;
  align-items: center;
}

.qo-line-fields {
  display: flex;
  gap: var(--space-md);
  flex: 1;
  align-items: flex-end;
}

.qo-field { display: flex; flex-direction: column; gap: var(--space-xs); }
.qo-field--sku { flex: 3; }
.qo-field--qty { flex: 1; }
.qo-label { font-size: var(--text-label-sm); font-weight: 600; }

.qo-input {
  height: 40px;
  padding: 0 var(--space-md);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  font-family: var(--font-family);
  font-size: var(--text-body-sm);
  outline: none;
}

.qo-input:focus { border-color: var(--secondary); box-shadow: 0 0 0 2px var(--secondary-fixed); }

.qo-line-remove {
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
  flex-shrink: 0;
}

.qo-line-remove:hover:not(:disabled) { background: var(--error-bg, #fef2f2); color: var(--error); }
.qo-line-remove:disabled { opacity: 0.3; cursor: not-allowed; }
.qo-line-remove .material-symbols-outlined { font-size: 16px; }

.qo-line-actions {
  display: flex;
  gap: var(--space-sm);
  padding-top: var(--space-sm);
  border-top: 1px solid var(--border);
}

.qo-add-line, .qo-csv-upload {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-sm) var(--space-md);
  border: 1px dashed var(--border);
  border-radius: var(--radius-lg);
  background: transparent;
  color: var(--muted);
  font-family: var(--font-family);
  font-size: var(--text-label-sm);
  font-weight: 600;
  cursor: pointer;
}

.qo-add-line:hover, .qo-csv-upload:hover { border-color: var(--secondary); color: var(--secondary); }
.qo-add-line .material-symbols-outlined, .qo-csv-upload .material-symbols-outlined { font-size: 16px; }
.qo-csv-input { display: none; }

.qo-messages {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.qo-message {
  padding: var(--space-md);
  border-radius: var(--radius-lg);
  font-size: var(--text-body-sm);
  border: 1px solid var(--border);
  background: var(--surface);
}

.qo-message--success { color: #16a34a; border-color: #bbf7d0; background: #f0fdf4; }
.qo-message--error { color: var(--error); border-color: #fecaca; background: #fef2f2; }

.qo-actions {
  display: flex;
  gap: var(--space-md);
}

.qo-submit {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  height: 44px;
  padding: 0 var(--space-lg);
  border: none;
  border-radius: var(--radius-lg);
  background: var(--primary);
  color: var(--on-primary);
  font-family: var(--font-family);
  font-size: var(--text-label-md);
  font-weight: 600;
  cursor: pointer;
}

.qo-submit:disabled { opacity: 0.5; cursor: not-allowed; }
.qo-submit .material-symbols-outlined { font-size: 20px; }

.qo-links {
  display: flex;
  gap: var(--space-lg);
}

.qo-link { color: var(--secondary); font-weight: 600; text-decoration: none; }
.qo-link:hover { text-decoration: underline; }
</style>
