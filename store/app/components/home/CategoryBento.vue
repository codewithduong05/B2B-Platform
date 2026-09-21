<script setup lang="ts">
interface ApiCategory {
  code: string
  name: string
  slug: string
  description: string
  sort_order: number
  is_active: boolean
}

interface CategoryBentoItem {
  title: string
  desc: string
  path: string
  icon: string
  count: string
}

const CATEGORY_ICONS: Record<string, string> = {
  'industrial-supplies': 'settings',
  'electrical-automation': 'bolt',
  'safety-ppe': 'health_and_safety',
  'packaging-logistics': 'local_shipping',
  'facility-maintenance': 'home_repair_service',
  'raw-materials': 'inventory_2',
}

const categories = ref<CategoryBentoItem[]>([])

onMounted(async () => {
  try {
    const data = await $fetch<{ items: ApiCategory[] }>('/api/catalog/categories', {
      query: { page: 1, page_size: 50 },
    })
    const topLevel = (data.items || []).filter(
      (c) => c.is_active && !c.description?.includes('sub'),
    )
    categories.value = topLevel.map((c) => ({
      title: c.name,
      desc: c.description || c.name,
      path: `/catalog?category=${c.slug}`,
      icon: CATEGORY_ICONS[c.slug] || 'category',
      count: '',
    }))
  } catch {
    categories.value = []
  }
})
</script>

<template>
  <section class="category-section">
    <div class="category-header">
      <h2 class="section-title">Browse by Category</h2>
      <a href="/catalog" class="category-explore-link">
        Explore All
        <span class="material-symbols-outlined">arrow_forward</span>
      </a>
    </div>
    <div class="category-grid">
      <a v-for="cat in categories" :key="cat.path" :href="cat.path" class="category-card">
        <div class="category-card-img">
          <span class="material-symbols-outlined category-card-icon">{{ cat.icon }}</span>
        </div>
        <div class="category-card-body">
          <h3 class="category-card-title">{{ cat.title }}</h3>
          <p class="category-card-desc">{{ cat.desc }}</p>
          <span class="category-card-count">{{ cat.count }}</span>
        </div>
      </a>
    </div>
  </section>
</template>

<style scoped>
.category-section {
  margin-bottom: var(--space-xl);
}

.category-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-lg);
}

.section-title {
  font-size: var(--text-headline-md);
  font-weight: 700;
  color: var(--on-surface);
  letter-spacing: var(--tracking-headline-md);
  margin: 0;
}

.category-explore-link {
  display: inline-flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-label-md);
  font-weight: 600;
  color: var(--secondary);
  text-decoration: none;
}

.category-explore-link:hover {
  color: var(--accent-hover);
}

.category-explore-link .material-symbols-outlined {
  font-size: 18px;
}

.category-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-lg);
}

.category-card {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  text-decoration: none;
  transition: box-shadow 0.15s ease;
}

.category-card:hover {
  box-shadow: var(--shadow-2);
}

.category-card-img {
  height: 176px;
  background-color: var(--surface-container-low);
  display: flex;
  align-items: center;
  justify-content: center;
}

.category-card-icon {
  font-size: 48px;
  color: var(--secondary);
  opacity: 0.6;
}

.category-card-body {
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.category-card-title {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
  margin: 0;
}

.category-card-desc {
  margin: 0;
  font-size: var(--text-body-sm);
  color: var(--muted);
  line-height: 1.5;
}

.category-card-count {
  font-size: var(--text-label-sm);
  font-weight: 500;
  color: var(--secondary);
}

@media (max-width: 768px) {
  .category-grid {
    grid-template-columns: 1fr;
  }
}
</style>
