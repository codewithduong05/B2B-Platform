<script setup lang="ts">
definePageMeta({ title: 'FAQ' })

interface FAQ {
  code: string
  question: string
  answer: string
  sort_order: number
}

const faqs = ref<FAQ[]>([])
const loading = ref(true)
const error = ref('')
const openIndex = ref<number | null>(null)

async function loadFaqs() {
  loading.value = true
  error.value = ''
  try {
    const data = await $fetch<{ items: FAQ[] }>('/api/faq')
    faqs.value = (data.items || []).sort((a, b) => a.sort_order - b.sort_order)
  } catch {
    error.value = 'Failed to load FAQs.'
  } finally {
    loading.value = false
  }
}

function toggle(index: number) {
  openIndex.value = openIndex.value === index ? null : index
}

onMounted(loadFaqs)
</script>

<template>
  <div class="faq-page">
    <nav class="faq-breadcrumb">
      <a href="/" class="faq-breadcrumb-link">Home</a>
      <span class="faq-breadcrumb-sep">/</span>
      <span class="faq-breadcrumb-current">FAQ</span>
    </nav>

    <h1 class="faq-title">Frequently Asked Questions</h1>

    <div v-if="loading" class="faq-loading">Loading FAQs...</div>
    <div v-else-if="error" class="faq-error">{{ error }}</div>
    <div v-else-if="faqs.length === 0" class="faq-empty">
      <span class="material-symbols-outlined faq-empty-icon">help_outline</span>
      <h2>No FAQs available</h2>
      <p>Check back later or contact support.</p>
    </div>
    <div v-else class="faq-list">
      <div v-for="(faq, index) in faqs" :key="faq.code" class="faq-item">
        <button class="faq-question" @click="toggle(index)">
          <span>{{ faq.question }}</span>
          <span class="material-symbols-outlined faq-chevron" :class="{ 'faq-chevron-open': openIndex === index }">
            expand_more
          </span>
        </button>
        <div v-if="openIndex === index" class="faq-answer">
          <p>{{ faq.answer }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.faq-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
  max-width: 800px;
}

.faq-breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.faq-breadcrumb-link { color: var(--muted); text-decoration: none; }
.faq-breadcrumb-link:hover { color: var(--secondary); }
.faq-breadcrumb-sep { color: var(--outline-variant); }
.faq-breadcrumb-current { font-weight: 600; color: var(--on-surface); }

.faq-title { margin: 0; font-size: var(--text-headline-lg); font-weight: 700; }

.faq-loading, .faq-error {
  padding: var(--space-2xl);
  text-align: center;
  color: var(--muted);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.faq-error { color: var(--error); }

.faq-empty {
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

.faq-empty-icon { font-size: 48px; color: var(--outline-variant); }
.faq-empty h2 { margin: 0; }
.faq-empty p { margin: 0; color: var(--muted); }

.faq-list { display: flex; flex-direction: column; gap: var(--space-sm); }

.faq-item {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.faq-question {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-md);
  padding: var(--space-md) var(--space-lg);
  border: none;
  background: transparent;
  font-family: var(--font-family);
  font-size: var(--text-body-md);
  font-weight: 600;
  color: var(--on-surface);
  text-align: left;
  cursor: pointer;
}

.faq-question:hover { background: var(--surface-container-low); }

.faq-chevron { transition: transform 0.2s ease; color: var(--outline-variant); }
.faq-chevron-open { transform: rotate(180deg); }

.faq-answer {
  padding: 0 var(--space-lg) var(--space-lg);
  border-top: 1px solid var(--border);
}

.faq-answer p { margin: var(--space-md) 0 0; color: var(--muted); line-height: 1.6; }
</style>
