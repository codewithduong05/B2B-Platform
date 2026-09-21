<script setup lang="ts">
import { ref } from 'vue'

const searchQuery = ref('')
const searchScope = ref('all')

const categories: Array<{ label: string; path: string }> = []
</script>

<template>
  <header class="store-header">
    <!-- Enterprise Top Bar -->
    <div class="header-context-bar">
      <div class="header-context-inner">
        <div class="header-context-left">
          <span class="material-symbols-outlined header-context-icon">domain</span>
          <span class="header-context-company"></span>
          <span class="header-context-id"></span>
          <span class="header-context-sep">|</span>
          <span class="badge badge-pending"></span>
          <span class="header-context-terms"></span>
        </div>
        <div class="header-context-right">
          <span class="header-context-label">Currency:</span>
          <span class="header-context-value">USD ($)</span>
          <span class="header-context-sep">|</span>
          <nav class="header-context-nav">
            <a href="/rfq">Request for Quote</a>
            <a href="/invoices">Invoices</a>
            <a href="/support">Help Desk</a>
          </nav>
        </div>
      </div>
    </div>

    <!-- Main Branding & Search -->
    <div class="header-main">
      <div class="header-main-inner">
        <!-- Logo -->
        <div class="header-brand">
          <img
            src="/logo.svg"
            alt="Atlas"
            class="header-logo"
            @error="($event.target as HTMLImageElement).style.display = 'none'"
          />
          <span class="header-brand-text">Atlas Procurement</span>
        </div>

        <!-- Search -->
        <div class="header-search">
          <div class="search-wrapper">
            <select v-model="searchScope" class="search-scope">
              <option value="all">All Products</option>
            </select>
            <span class="search-divider"></span>
            <span class="material-symbols-outlined search-icon">search</span>
            <input
              v-model="searchQuery"
              type="text"
              class="search-input"
              placeholder="Search catalog by SKU, manufacturer part #, or UNSPSC..."
            />
          </div>
        </div>

        <!-- User Actions -->
        <div class="header-actions">
          <a href="/quick-order" class="header-action-btn">
            <span class="material-symbols-outlined">bolt</span>
            <span class="header-action-label">Quick Order</span>
          </a>
          <a href="/rfq" class="header-icon-btn" title="RFQ Queue">
            <span class="material-symbols-outlined">request_quote</span>
          </a>
          <a href="/cart" class="header-icon-btn" title="Cart">
            <span class="material-symbols-outlined">shopping_cart</span>
          </a>
          <span class="header-divider"></span>
          <div class="header-user">
            <div class="header-avatar">
              <span class="material-symbols-outlined">person</span>
            </div>
            <div class="header-user-info">
              <span class="header-user-name">User</span>
              <span class="header-user-role">
                <span class="material-symbols-outlined">verified</span>
                Buyer
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Category Navigation -->
    <div class="header-categories">
      <div class="header-categories-inner">
        <nav class="header-categories-nav">
          <a
            v-for="cat in categories"
            :key="cat.path"
            :href="cat.path"
            class="header-category-link"
          >
            {{ cat.label }}
          </a>
        </nav>
      </div>
    </div>
  </header>
</template>

<style scoped>
.store-header {
  position: sticky;
  top: 0;
  z-index: 50;
  background-color: var(--surface);
  box-shadow: 0 1px 8px rgba(0, 0, 0, 0.04);
}

/* ── Context Bar ── */
.header-context-bar {
  height: 32px;
  background-color: var(--surface-container-low);
  border-bottom: 1px solid var(--border);
}

.header-context-inner {
  max-width: 1440px;
  margin: 0 auto;
  height: 100%;
  padding: 0 var(--margin);
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: var(--text-label-sm);
  line-height: var(--line-label-sm);
  color: var(--muted);
}

.header-context-left {
  display: flex;
  align-items: center;
  gap: var(--space-lg);
}

.header-context-icon {
  font-size: 14px;
  color: var(--secondary);
}

.header-context-company {
  font-weight: 500;
  color: var(--on-surface);
}

.header-context-id {
  color: var(--muted);
}

.header-context-sep {
  color: var(--outline-variant);
}

.header-context-terms {
  color: var(--muted);
}

.header-context-right {
  display: flex;
  align-items: center;
  gap: var(--space-lg);
}

.header-context-label {
  color: var(--muted);
}

.header-context-value {
  font-weight: 500;
  color: var(--on-surface);
}

.header-context-nav {
  display: flex;
  align-items: center;
  gap: var(--space-md);
}

.header-context-nav a {
  color: var(--muted);
  text-decoration: none;
  transition: color 0.15s ease;
}

.header-context-nav a:hover {
  color: var(--secondary);
}

/* ── Main Bar ── */
.header-main {
  height: 64px;
}

.header-main-inner {
  max-width: 1440px;
  margin: 0 auto;
  height: 100%;
  padding: 0 var(--margin);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-xl);
}

/* ── Brand ── */
.header-brand {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  flex-shrink: 0;
}

.header-logo {
  height: 32px;
  width: auto;
}

.header-brand-text {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  letter-spacing: var(--tracking-headline-sm);
  color: var(--on-surface);
  text-transform: uppercase;
}

/* ── Search ── */
.header-search {
  flex: 1;
  max-width: 640px;
}

.search-wrapper {
  display: flex;
  align-items: center;
  background-color: var(--surface-container-low);
  border-radius: var(--radius-lg);
  padding: 0 var(--space-sm);
}

.search-scope {
  appearance: none;
  background: transparent;
  border: none;
  padding: var(--space-xs) var(--space-md);
  font-family: var(--font-family);
  font-size: var(--text-label-md);
  color: var(--on-surface);
  cursor: pointer;
  outline: none;
}

.search-divider {
  width: 1px;
  height: 20px;
  background-color: var(--outline-variant);
}

.search-icon {
  font-size: 20px;
  color: var(--muted);
  padding: 0 var(--space-xs);
}

.search-input {
  flex: 1;
  background: transparent;
  border: none;
  padding: var(--space-xs) 0;
  font-family: var(--font-family);
  font-size: var(--text-body-md);
  color: var(--on-surface);
  outline: none;
}

.search-input::placeholder {
  color: #94a3b8;
}

/* ── Actions ── */
.header-actions {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  flex-shrink: 0;
}

.header-action-btn {
  display: inline-flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-xs) var(--space-md);
  border-radius: var(--radius);
  background-color: var(--surface-container);
  color: var(--on-surface);
  font-size: var(--text-label-md);
  text-decoration: none;
  transition: background-color 0.15s ease;
}

.header-action-btn:hover {
  background-color: var(--surface-container-high);
}

.header-action-label {
  font-weight: 500;
}

.header-icon-btn {
  position: relative;
  display: flex;
  align-items: center;
  padding: var(--space-xs);
  border-radius: var(--radius);
  color: var(--muted);
  text-decoration: none;
  transition:
    color 0.15s ease,
    background-color 0.15s ease;
}

.header-icon-btn:hover {
  color: var(--on-surface);
  background-color: var(--surface-container-low);
}

.header-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  min-width: 16px;
  height: 16px;
  border-radius: var(--radius-full);
  font-size: 10px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
}

.header-badge-primary {
  background-color: var(--secondary);
  color: var(--on-secondary);
}

.header-badge-secondary {
  background-color: var(--accent);
  color: var(--on-primary);
}

.header-divider {
  width: 1px;
  height: 24px;
  background-color: var(--surface-container-high);
}

.header-user {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
}

.header-avatar {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-full);
  background-color: var(--accent);
  color: var(--on-primary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.header-avatar .material-symbols-outlined {
  font-size: 20px;
}

.header-user-info {
  display: flex;
  flex-direction: column;
}

.header-user-name {
  font-size: var(--text-label-md);
  font-weight: 600;
  color: var(--on-surface);
}

.header-user-role {
  display: flex;
  align-items: center;
  gap: 2px;
  font-size: var(--text-label-sm);
  color: var(--secondary);
}

.header-user-role .material-symbols-outlined {
  font-size: 12px;
}

/* ── Categories ── */
.header-categories {
  border-top: 1px solid var(--border);
}

.header-categories-inner {
  max-width: 1440px;
  margin: 0 auto;
  padding: 0 var(--margin);
}

.header-categories-nav {
  display: flex;
  align-items: center;
  gap: var(--space-xl);
  overflow-x: auto;
  white-space: nowrap;
  padding: var(--space-xs) 0;
  scrollbar-width: none;
}

.header-categories-nav::-webkit-scrollbar {
  display: none;
}

.header-category-link {
  font-size: var(--text-label-md);
  color: var(--muted);
  text-decoration: none;
  transition: color 0.15s ease;
  flex-shrink: 0;
}

.header-category-link:hover {
  color: var(--secondary);
  font-weight: 600;
}

/* ── Responsive ── */
@media (max-width: 768px) {
  .header-context-right,
  .header-action-label,
  .header-user-info {
    display: none;
  }
}
</style>
