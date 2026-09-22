<template>
  <div class="admin-layout">
    <aside class="admin-sidebar">
      <div class="sidebar-header">
        <span class="logo-text">ATLAS</span>
        <span class="logo-badge">ADMIN</span>
      </div>
      <nav class="sidebar-nav">
        <NuxtLink
          v-for="item in navItems"
          :key="item.route"
          :to="item.route"
          class="nav-item"
          :class="{ active: route.path === item.route }"
        >
          <span class="material-symbols-outlined nav-icon">{{ item.icon }}</span>
          <span class="nav-label">{{ item.title }}</span>
        </NuxtLink>
      </nav>
    </aside>
    <div class="admin-main">
      <header class="admin-topbar">
        <div class="topbar-left">
          <button class="menu-toggle" @click="sidebarOpen = !sidebarOpen">
            <span class="material-symbols-outlined">menu</span>
          </button>
          <div class="search-box">
            <span class="material-symbols-outlined search-icon">search</span>
            <input type="text" placeholder="Search..." class="search-input" />
          </div>
        </div>
        <div class="topbar-right">
          <span class="topbar-label">CLUSTER: SGN-DC1</span>
        </div>
      </header>
      <main class="admin-content">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const sidebarOpen = ref(false)

const navItems = [
  { route: '/dashboard', title: 'Dashboard', icon: 'dashboard' },
  { route: '/products', title: 'Products', icon: 'inventory_2' },
  { route: '/orders', title: 'Orders', icon: 'receipt_long' },
  { route: '/customers', title: 'Customers', icon: 'people' },
  { route: '/inventory', title: 'Inventory', icon: 'warehouse' },
  { route: '/settings', title: 'Settings', icon: 'settings' },
]
</script>

<style scoped>
.admin-layout {
  display: flex;
  min-height: 100vh;
  background: var(--bg);
  color: var(--text);
}

.admin-sidebar {
  width: 256px;
  background: var(--surface);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  z-index: 40;
}

.sidebar-header {
  padding: 1rem 1.25rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border-bottom: 1px solid var(--border);
}

.logo-text {
  font-weight: 700;
  font-size: 1.125rem;
  letter-spacing: -0.02em;
}

.logo-badge {
  font-size: 0.625rem;
  font-weight: 600;
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
  background: var(--primary-container);
  color: var(--on-primary-container);
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.sidebar-nav {
  flex: 1;
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 0.75rem;
  border-radius: 8px;
  text-decoration: none;
  color: var(--text-secondary);
  font-size: 0.875rem;
  font-weight: 500;
  transition: background 0.15s;
}

.nav-item:hover {
  background: var(--surface-container-low);
}

.nav-item.active {
  background: var(--primary-container);
  color: var(--on-primary-container);
  font-weight: 600;
}

.nav-icon {
  font-size: 20px;
}

.admin-main {
  flex: 1;
  margin-left: 256px;
  display: flex;
  flex-direction: column;
}

.admin-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1.5rem;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
  z-index: 30;
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.menu-toggle {
  display: none;
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text);
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 0.75rem;
  font-size: 18px;
  color: var(--muted);
}

.search-input {
  width: 320px;
  padding: 0.5rem 0.75rem 0.5rem 2.5rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-container-low);
  font-size: 0.875rem;
  outline: none;
}

.search-input:focus {
  background: var(--surface);
  border-color: var(--secondary);
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.topbar-label {
  font-size: 0.75rem;
  color: var(--muted);
  padding: 0.25rem 0.5rem;
  background: var(--surface-container-low);
  border-radius: 4px;
}

.admin-content {
  flex: 1;
  padding: 1.5rem;
  max-width: 1680px;
  width: 100%;
  margin: 0 auto;
}

@media (max-width: 1024px) {
  .admin-sidebar {
    transform: translateX(-100%);
    transition: transform 0.2s;
  }

  .admin-sidebar.open {
    transform: translateX(0);
  }

  .admin-main {
    margin-left: 0;
  }

  .menu-toggle {
    display: flex;
  }

  .search-input {
    width: 200px;
  }
}
</style>
