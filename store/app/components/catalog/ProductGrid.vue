<script setup lang="ts">
import type { CatalogProduct } from '~~/shared/catalog'

defineProps<{ products: CatalogProduct[] }>()
</script>

<template>
  <div class="product-grid">
    <CatalogProductCard v-for="product in products" :key="product.id" :product="product" />
  </div>
</template>

<style scoped>
.product-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-md);
}

@media (max-width: 1024px) {
  .product-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 640px) {
  .product-grid {
    grid-template-columns: 1fr;
  }
}
</style>
