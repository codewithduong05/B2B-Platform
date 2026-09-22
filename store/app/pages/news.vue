<script setup lang="ts">
definePageMeta({ title: 'News' })

interface Article {
  code: string
  title: string
  excerpt: string
  slug: string
  author: string
  published_at: string
}

const articles = ref<Article[]>([])
const loading = ref(true)
const error = ref('')

async function loadArticles() {
  loading.value = true
  error.value = ''
  try {
    const data = await $fetch<{ items: Article[] }>('/api/news')
    articles.value = data.items || []
  } catch {
    error.value = 'Failed to load articles.'
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}

onMounted(loadArticles)
</script>

<template>
  <div class="news-page">
    <nav class="news-breadcrumb">
      <a href="/" class="news-breadcrumb-link">Home</a>
      <span class="news-breadcrumb-sep">/</span>
      <span class="news-breadcrumb-current">News</span>
    </nav>

    <h1 class="news-title">News &amp; Updates</h1>

    <div v-if="loading" class="news-loading">Loading articles...</div>
    <div v-else-if="error" class="news-error">{{ error }}</div>
    <div v-else-if="articles.length === 0" class="news-empty">
      <span class="material-symbols-outlined news-empty-icon">article</span>
      <h2>No articles yet</h2>
      <p>Check back later for updates.</p>
    </div>
    <div v-else class="news-list">
      <article v-for="article in articles" :key="article.code" class="news-card">
        <h2 class="news-card-title">{{ article.title }}</h2>
        <div class="news-card-meta">
          <span v-if="article.author">{{ article.author }}</span>
          <span v-if="article.published_at">{{ formatDate(article.published_at) }}</span>
        </div>
        <p class="news-card-excerpt">{{ article.excerpt }}</p>
      </article>
    </div>
  </div>
</template>

<style scoped>
.news-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

.news-breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.news-breadcrumb-link { color: var(--muted); text-decoration: none; }
.news-breadcrumb-link:hover { color: var(--secondary); }
.news-breadcrumb-sep { color: var(--outline-variant); }
.news-breadcrumb-current { font-weight: 600; color: var(--on-surface); }

.news-title { margin: 0; font-size: var(--text-headline-lg); font-weight: 700; }

.news-loading, .news-error {
  padding: var(--space-2xl);
  text-align: center;
  color: var(--muted);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.news-error { color: var(--error); }

.news-empty {
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

.news-empty-icon { font-size: 48px; color: var(--outline-variant); }
.news-empty h2 { margin: 0; }
.news-empty p { margin: 0; color: var(--muted); }

.news-list { display: flex; flex-direction: column; gap: var(--space-md); }

.news-card {
  padding: var(--space-lg);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.news-card-title { margin: 0 0 var(--space-xs); font-size: var(--text-headline-sm); font-weight: 700; }
.news-card-meta { display: flex; gap: var(--space-md); font-size: var(--text-label-sm); color: var(--muted); margin-bottom: var(--space-sm); }
.news-card-excerpt { margin: 0; color: var(--muted); font-size: var(--text-body-md); line-height: 1.6; }
</style>
