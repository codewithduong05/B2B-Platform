<template>
  <section class="compliance-strip">
    <div class="compliance-inner">
      <div class="compliance-item">
        <span class="material-symbols-outlined compliance-icon">workspace_premium</span>
        <span class="compliance-text">ISO 9001:2015 Certified</span>
      </div>
      <div class="compliance-sep"></div>
      <div class="compliance-item">
        <span class="material-symbols-outlined compliance-icon">eco</span>
        <span class="compliance-text">REACH &amp; RoHS Compliant</span>
      </div>
      <div class="compliance-sep"></div>
      <div class="compliance-item">
        <span class="material-symbols-outlined compliance-icon">qr_code_2</span>
        <span class="compliance-text">Full Lot Traceability</span>
      </div>
      <div class="compliance-sep"></div>
      <div class="compliance-item">
        <span class="material-symbols-outlined compliance-icon">verified_user</span>
        <span class="compliance-text">SOC 2 Type II Audited</span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.compliance-strip {
  margin-bottom: var(--space-xl);
}

.compliance-inner {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-lg);
  padding: var(--space-md) var(--space-lg);
  background-color: var(--surface-container-low);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  flex-wrap: wrap;
}

.compliance-item {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
}

.compliance-icon {
  font-size: 18px;
  color: var(--secondary);
}

.compliance-text {
  font-size: var(--text-label-md);
  font-weight: 500;
  color: var(--muted);
}

.compliance-sep {
  width: 1px;
  height: 16px;
  background-color: var(--border);
}

@media (max-width: 768px) {
  .compliance-inner {
    gap: var(--space-md);
  }

  .compliance-sep {
    display: none;
  }
}
</style>
