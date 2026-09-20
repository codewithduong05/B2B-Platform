<script setup lang="ts">
import type { ProductImage } from '~~/shared/product'

defineProps<{ images: ProductImage[] }>()

const activeIndex = ref(0)

function selectImage(index: number) {
  activeIndex.value = index
}
</script>

<template>
  <div class="image-gallery">
    <div class="image-main">
      <div class="image-main-viewport">
        <span class="material-symbols-outlined image-main-placeholder">inventory_2</span>
        <span class="image-overlay-badge image-overlay-cad">
          <span class="material-symbols-outlined">engineering</span>
          3D CAD Validated
        </span>
        <span class="image-overlay-badge image-overlay-temp">Class H 180°C Tested</span>
      </div>
      <div class="image-main-controls">
        <button class="image-control-btn" title="Zoom In">
          <span class="material-symbols-outlined">zoom_in</span>
        </button>
        <button class="image-control-btn" title="Full Dimensional Diagram">
          <span class="material-symbols-outlined">aspect_ratio</span>
        </button>
        <button class="image-control-btn" title="360 Interactive Turntable">
          <span class="material-symbols-outlined">360</span>
        </button>
      </div>
    </div>
    <div class="image-thumbs">
      <button
        v-for="(img, i) in images"
        :key="i"
        class="image-thumb"
        :class="{ 'image-thumb-active': i === activeIndex }"
        @click="selectImage(i)"
      >
        <span class="material-symbols-outlined image-thumb-icon">inventory_2</span>
        <span class="image-thumb-label">{{ img.label }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.image-gallery {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: var(--space-lg);
}

.image-main {
  position: relative;
}

.image-main-viewport {
  width: 100%;
  aspect-ratio: 4 / 3;
  background-color: var(--surface-container-low);
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.image-main-placeholder {
  font-size: 64px;
  color: var(--outline-variant);
}

.image-overlay-badge {
  position: absolute;
  top: 12px;
  left: 12px;
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  padding: 4px 10px;
  border-radius: var(--radius);
  font-size: var(--text-label-sm);
  font-weight: 600;
  letter-spacing: var(--tracking-label-sm);
  text-transform: uppercase;
}

.image-overlay-cad {
  background-color: var(--accent);
  color: var(--on-primary);
}

.image-overlay-cad .material-symbols-outlined {
  font-size: 14px;
}

.image-overlay-temp {
  position: absolute;
  top: 44px;
  left: 12px;
  background-color: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(4px);
  color: var(--on-surface);
  font-weight: 500;
  text-transform: none;
  letter-spacing: normal;
}

.image-main-controls {
  position: absolute;
  bottom: 12px;
  right: 12px;
  display: flex;
  gap: 6px;
  background-color: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(4px);
  padding: 6px;
  border-radius: var(--radius-lg);
}

.image-control-btn {
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
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
}

.image-control-btn:hover {
  background-color: var(--surface-container);
  color: var(--on-surface);
}

/* ── Thumbnails ── */
.image-thumbs {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-md);
  margin-top: var(--space-md);
}

.image-thumb {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-sm);
  border: 2px solid transparent;
  border-radius: var(--radius);
  background-color: var(--surface-container-low);
  cursor: pointer;
  transition: border-color 0.15s ease;
}

.image-thumb:hover {
  border-color: var(--secondary);
}

.image-thumb-active {
  border-color: var(--secondary);
}

.image-thumb-icon {
  font-size: 24px;
  color: var(--outline-variant);
}

.image-thumb-label {
  font-size: var(--text-label-sm);
  font-weight: 500;
  color: var(--muted);
}
</style>
