<script setup lang="ts">
definePageMeta({ title: 'Order Confirmed' })

const route = useRoute()
const orderCode = route.params.code as string
</script>

<template>
  <div class="confirm-page">
    <div class="confirm-card">
      <span class="material-symbols-outlined confirm-icon">check_circle</span>
      <h1 class="confirm-title">Order Placed Successfully</h1>
      <p class="confirm-desc">
        Your order <strong class="confirm-code">{{ orderCode }}</strong> has been submitted.
        You will receive a confirmation once it is processed.
      </p>

      <div class="confirm-actions">
        <a :href="`/order/${orderCode}`" class="confirm-btn confirm-btn-primary">
          <span class="material-symbols-outlined">receipt_long</span>
          View Order Details
        </a>
        <a href="/orders" class="confirm-btn confirm-btn-secondary">
          <span class="material-symbols-outlined">list_alt</span>
          Order History
        </a>
        <a href="/catalog" class="confirm-btn confirm-btn-tertiary">
          <span class="material-symbols-outlined">storefront</span>
          Continue Shopping
        </a>
      </div>
    </div>
  </div>
</template>

<style scoped>
.confirm-page {
  display: flex;
  justify-content: center;
  padding: var(--space-2xl) 0;
}

.confirm-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-lg);
  max-width: 480px;
  padding: var(--space-2xl);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  text-align: center;
}

.confirm-icon {
  font-size: 64px;
  color: var(--secondary);
}

.confirm-title {
  margin: 0;
  font-size: var(--text-headline-lg);
  font-weight: 700;
  color: var(--on-surface);
}

.confirm-desc {
  margin: 0;
  font-size: var(--text-body-md);
  color: var(--muted);
  line-height: 1.6;
}

.confirm-code {
  font-family: monospace;
  color: var(--on-surface);
  background: var(--surface-container-high);
  padding: 2px 8px;
  border-radius: var(--radius);
}

.confirm-actions {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
  width: 100%;
  margin-top: var(--space-md);
}

.confirm-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-sm);
  height: 44px;
  border-radius: var(--radius-lg);
  text-decoration: none;
  font-weight: 600;
  font-size: var(--text-label-md);
}

.confirm-btn .material-symbols-outlined { font-size: 20px; }

.confirm-btn-primary {
  background: var(--primary);
  color: var(--on-primary);
}

.confirm-btn-secondary {
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--on-surface);
}

.confirm-btn-tertiary {
  background: transparent;
  color: var(--secondary);
}
</style>
