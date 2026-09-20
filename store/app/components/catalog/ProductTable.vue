<script setup lang="ts">
import type { CatalogProduct } from '~~/shared/catalog'

defineProps<{ products: CatalogProduct[] }>()
</script>

<template>
  <div class="product-table-wrapper">
    <table class="product-table">
      <thead>
        <tr>
          <th class="product-th product-th-check">
            <input type="checkbox" class="table-checkbox" />
          </th>
          <th class="product-th">SKU / Item</th>
          <th class="product-th">Supplier</th>
          <th class="product-th">Specs / Standards</th>
          <th class="product-th">Availability</th>
          <th class="product-th product-th-right">Contract Price</th>
          <th class="product-th product-th-center">Action</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="p in products" :key="p.id" class="product-tr">
          <td class="product-td product-td-check">
            <input type="checkbox" class="table-checkbox" />
          </td>
          <td class="product-td">
            <span class="product-td-sku">{{ p.sku }}</span>
            <span class="product-td-name">{{ p.name }}</span>
          </td>
          <td class="product-td">
            <span class="product-td-supplier">{{ p.supplier }}</span>
          </td>
          <td class="product-td product-td-specs">{{ p.specs }}</td>
          <td class="product-td">
            <span :class="p.lowStock ? 'product-td-stock-low' : 'product-td-stock'">
              {{ p.stockLabel }}
            </span>
          </td>
          <td class="product-td product-td-price">${{ p.price.toFixed(2) }}</td>
          <td class="product-td product-td-action">
            <NuxtLink :to="`/product/${p.sku}`" class="table-action-btn"> Requisition </NuxtLink>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.product-table-wrapper {
  overflow-x: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.product-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-body-sm);
}

.product-th {
  padding: var(--space-sm) var(--space-md);
  background-color: var(--surface-container-low);
  font-size: var(--text-label-sm);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: var(--tracking-label-sm);
  color: var(--muted);
  text-align: left;
  border-bottom: 1px solid var(--border-strong);
}

.product-th-check {
  width: 40px;
}

.product-th-right {
  text-align: right;
}

.product-th-center {
  text-align: center;
}

.product-tr {
  border-bottom: 1px solid var(--border);
  transition: background-color 0.1s ease;
}

.product-tr:hover {
  background-color: var(--surface-container-low);
}

.product-td {
  padding: var(--space-sm) var(--space-md);
  vertical-align: middle;
}

.product-td-check {
  width: 40px;
  text-align: center;
}

.table-checkbox {
  width: 16px;
  height: 16px;
  accent-color: var(--secondary);
}

.product-td-sku {
  display: block;
  font-size: var(--text-headline-sm);
  font-weight: 600;
  color: var(--on-surface);
}

.product-td-name {
  display: block;
  font-size: var(--text-body-sm);
  color: var(--muted);
}

.product-td-supplier {
  padding: 2px 8px;
  border-radius: var(--radius);
  background-color: var(--surface-container);
  font-size: var(--text-label-sm);
  font-weight: 600;
  color: var(--secondary);
}

.product-td-specs {
  color: var(--muted);
}

.product-td-stock {
  font-size: var(--text-tabular-sm);
  font-weight: 600;
  color: var(--secondary);
}

.product-td-stock-low {
  font-size: var(--text-tabular-sm);
  font-weight: 600;
  color: var(--error);
}

.product-td-price {
  text-align: right;
  font-size: var(--text-tabular-md);
  font-weight: 700;
  color: var(--on-surface);
  font-variant-numeric: tabular-nums;
}

.product-td-action {
  text-align: center;
}

.table-action-btn {
  display: inline-flex;
  align-items: center;
  padding: 4px 12px;
  border-radius: var(--radius);
  background-color: var(--accent);
  color: var(--on-primary);
  font-size: var(--text-label-sm);
  font-weight: 600;
  text-decoration: none;
  transition: background-color 0.15s ease;
}

.table-action-btn:hover {
  background-color: var(--accent-hover);
}
</style>
