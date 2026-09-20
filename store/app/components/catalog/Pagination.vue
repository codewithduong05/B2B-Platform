<script setup lang="ts">
const props = defineProps<{
  currentPage: number
  totalPages: number
  total: number
  perPage: number
  start: number
  end: number
}>()

const emit = defineEmits<{
  (e: 'page', page: number): void
}>()

const jumpValue = ref(props.currentPage)

function goToPage(page: number) {
  if (page >= 1 && page <= props.totalPages) {
    emit('page', page)
    jumpValue.value = page
  }
}

function handleJump() {
  goToPage(jumpValue.value)
}

watch(
  () => props.currentPage,
  (val) => {
    jumpValue.value = val
  },
)
</script>

<template>
  <div class="pagination-bar">
    <div class="pagination-info">
      <span>
        Showing <strong>{{ start }}</strong> - <strong>{{ end }}</strong> of
        <strong>{{ total }}</strong> verified enterprise products
      </span>
    </div>

    <div class="pagination-controls">
      <button class="pagination-btn" :disabled="currentPage <= 1" @click="goToPage(1)">
        <span class="material-symbols-outlined">first_page</span>
      </button>
      <button
        class="pagination-btn"
        :disabled="currentPage <= 1"
        @click="goToPage(currentPage - 1)"
      >
        <span class="material-symbols-outlined">chevron_left</span>
      </button>

      <template v-for="page in totalPages" :key="page">
        <button
          v-if="page === 1 || page === totalPages || Math.abs(page - currentPage) <= 1"
          class="pagination-page"
          :class="{ 'pagination-page-active': page === currentPage }"
          @click="goToPage(page)"
        >
          {{ page }}
        </button>
        <span v-else-if="Math.abs(page - currentPage) === 2" class="pagination-ellipsis">
          ...
        </span>
      </template>

      <button
        class="pagination-btn"
        :disabled="currentPage >= totalPages"
        @click="goToPage(currentPage + 1)"
      >
        <span class="material-symbols-outlined">chevron_right</span>
      </button>
      <button
        class="pagination-btn"
        :disabled="currentPage >= totalPages"
        @click="goToPage(totalPages)"
      >
        <span class="material-symbols-outlined">last_page</span>
      </button>
    </div>

    <div class="pagination-jump">
      <span>Go to page:</span>
      <input
        v-model.number="jumpValue"
        type="number"
        :min="1"
        :max="totalPages"
        class="pagination-jump-input"
        @keydown.enter="handleJump"
      />
      <button class="pagination-jump-btn" @click="handleJump">Go</button>
    </div>
  </div>
</template>

<style scoped>
.pagination-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-md);
  padding: var(--space-md);
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  flex-wrap: wrap;
}

.pagination-info {
  font-size: var(--text-body-sm);
  color: var(--muted);
}

.pagination-info strong {
  color: var(--on-surface);
}

.pagination-controls {
  display: flex;
  align-items: center;
  gap: 4px;
}

.pagination-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: var(--radius);
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.pagination-btn:hover:not(:disabled) {
  background-color: var(--surface-container-low);
  color: var(--on-surface);
}

.pagination-btn:disabled {
  opacity: 0.4;
  cursor: default;
}

.pagination-page {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: var(--radius);
  background: transparent;
  font-family: var(--font-family);
  font-size: var(--text-tabular-sm);
  font-weight: 500;
  color: var(--on-surface);
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.pagination-page:hover {
  background-color: var(--surface-container-low);
}

.pagination-page-active {
  background-color: var(--accent);
  color: var(--on-primary);
  font-weight: 700;
}

.pagination-page-active:hover {
  background-color: var(--accent-hover);
}

.pagination-ellipsis {
  width: 32px;
  text-align: center;
  color: var(--muted);
}

.pagination-jump {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-body-sm);
  color: var(--muted);
}

.pagination-jump-input {
  width: 48px;
  height: 32px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background-color: var(--surface-container-low);
  text-align: center;
  font-family: var(--font-family);
  font-size: var(--text-tabular-sm);
  color: var(--on-surface);
  outline: none;
}

.pagination-jump-input:focus {
  border-color: var(--secondary);
}

.pagination-jump-btn {
  padding: 4px 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background-color: var(--surface-container);
  font-family: var(--font-family);
  font-size: var(--text-label-sm);
  font-weight: 600;
  color: var(--on-surface);
  cursor: pointer;
}

.pagination-jump-btn:hover {
  background-color: var(--surface-container-high);
}

@media (max-width: 768px) {
  .pagination-bar {
    flex-direction: column;
    align-items: center;
  }
}
</style>
