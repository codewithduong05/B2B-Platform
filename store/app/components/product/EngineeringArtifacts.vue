<script setup lang="ts">
import type { Artifact } from '~~/shared/product'

defineProps<{ artifacts: Artifact[] }>()
</script>

<template>
  <div class="artifacts-box">
    <div class="artifacts-header">
      <div class="artifacts-header-left">
        <span class="material-symbols-outlined artifacts-header-icon">folder_special</span>
        <span class="artifacts-header-title">Engineering &amp; CAD Artifacts</span>
      </div>
      <span class="artifacts-revision">Vault Rev: 2025.2-LTC</span>
    </div>
    <div class="artifacts-grid">
      <a v-for="artifact in artifacts" :key="artifact.title" href="#" class="artifact-card">
        <div class="artifact-card-top">
          <span
            class="material-symbols-outlined artifact-icon"
            :class="`artifact-icon-${artifact.iconColor}`"
          >
            {{ artifact.icon }}
          </span>
          <span class="material-symbols-outlined artifact-download">download</span>
        </div>
        <div class="artifact-card-bottom">
          <span class="artifact-title">{{ artifact.title }}</span>
          <span class="artifact-desc">{{ artifact.description }}</span>
        </div>
      </a>
    </div>
  </div>
</template>

<style scoped>
.artifacts-box {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: var(--space-lg);
}

.artifacts-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-sm) var(--space-md);
  background-color: var(--surface-container-low);
  border-radius: var(--radius-lg);
  margin-bottom: var(--space-md);
}

.artifacts-header-left {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.artifacts-header-icon {
  font-size: 22px;
  color: var(--secondary);
}

.artifacts-header-title {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
}

.artifacts-revision {
  font-size: var(--text-label-sm);
  font-weight: 500;
  color: var(--muted);
}

.artifacts-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-md);
}

.artifact-card {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: var(--space-md);
  border-radius: var(--radius-lg);
  background-color: var(--surface-container-low);
  text-decoration: none;
  transition: background-color 0.15s ease;
  min-height: 120px;
}

.artifact-card:hover {
  background-color: var(--surface-container-high);
}

.artifact-card-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.artifact-icon {
  font-size: 28px;
}

.artifact-icon-error {
  color: var(--error);
}

.artifact-icon-secondary {
  color: var(--secondary);
}

.artifact-download {
  font-size: 18px;
  color: var(--muted);
  transition: color 0.15s ease;
}

.artifact-card:hover .artifact-download {
  color: var(--secondary);
}

.artifact-card-bottom {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.artifact-title {
  font-size: var(--text-label-md);
  font-weight: 700;
  color: var(--on-surface);
  line-height: 1.3;
}

.artifact-desc {
  font-size: var(--text-body-sm);
  color: var(--muted);
}

@media (max-width: 768px) {
  .artifacts-grid {
    grid-template-columns: 1fr;
  }
}
</style>
