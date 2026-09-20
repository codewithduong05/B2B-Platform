<template>
  <section class="contract-banner">
    <div class="contract-banner-inner">
      <div class="contract-banner-left">
        <span class="material-symbols-outlined contract-banner-icon">verified_user</span>
        <div class="contract-banner-text">
          <h3 class="contract-banner-title">Master Catalogues &amp; Contract Compliance</h3>
          <p class="contract-banner-desc">
            All listed SKUs are mapped against your enterprise Master Service Agreement (#MA-77109).
            Pricing, lead times, and compliance certifications are synced in real-time with Tier 1
            vendor feeds.
          </p>
        </div>
      </div>
      <div class="contract-banner-actions">
        <button class="button button-secondary">
          <span class="material-symbols-outlined">receipt_long</span>
          Audit Rate Sheet
        </button>
        <a href="/contracts" class="contract-banner-link"> View Agreement &rarr; </a>
      </div>
    </div>
  </section>
</template>

<style scoped>
.contract-banner {
  margin-bottom: var(--space-xl);
}

.contract-banner-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-lg);
  padding: var(--space-lg);
  background-color: var(--surface-container-low);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.contract-banner-left {
  display: flex;
  align-items: flex-start;
  gap: var(--space-md);
}

.contract-banner-icon {
  font-size: 28px;
  color: var(--secondary);
  flex-shrink: 0;
  margin-top: 2px;
}

.contract-banner-text {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.contract-banner-title {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
  margin: 0;
}

.contract-banner-desc {
  margin: 0;
  font-size: var(--text-body-sm);
  color: var(--muted);
  max-width: 640px;
  line-height: 1.6;
}

.contract-banner-actions {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: var(--space-sm);
  flex-shrink: 0;
}

.contract-banner-link {
  font-size: var(--text-label-md);
  font-weight: 600;
  color: var(--secondary);
  text-decoration: none;
}

.contract-banner-link:hover {
  color: var(--accent-hover);
}

@media (max-width: 768px) {
  .contract-banner-inner {
    flex-direction: column;
    align-items: flex-start;
  }

  .contract-banner-actions {
    flex-direction: row;
    align-items: center;
  }
}
</style>
