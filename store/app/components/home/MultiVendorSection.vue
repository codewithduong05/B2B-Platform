<template>
  <section class="vendor-section">
    <h2 class="section-title">Multi-Vendor Engine</h2>
    <div class="vendor-grid">
      <div class="vendor-card">
        <span class="material-symbols-outlined vendor-icon">call_split</span>
        <h3 class="vendor-card-title">Split Routing</h3>
        <p class="vendor-card-desc">
          Automatically distribute line items across approved vendors based on contract tier, lead
          time, and regional availability.
        </p>
        <span class="vendor-card-stat">
          <span class="vendor-stat-num">--</span> vendors active
        </span>
      </div>
      <div class="vendor-card">
        <span class="material-symbols-outlined vendor-icon">receipt_long</span>
        <h3 class="vendor-card-title">Unified Billing</h3>
        <p class="vendor-card-desc">
          Consolidate multi-supplier invoices into a single reconciliation stream with automated
          3-way matching against POs.
        </p>
        <span class="vendor-card-stat">
          <span class="vendor-stat-num">--</span> processed this quarter
        </span>
      </div>
      <div class="vendor-card">
        <span class="material-symbols-outlined vendor-icon">sync_alt</span>
        <h3 class="vendor-card-title">cXML / PunchOut</h3>
        <p class="vendor-card-desc">
          Pre-configured PunchOut catalog connections with SAP Ariba, Coupa, and Oracle iProcurement
          for seamless procurement integration.
        </p>
        <span class="vendor-card-stat">
          <span class="vendor-stat-num">--</span> integrations live
        </span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.vendor-section {
  margin-bottom: var(--space-xl);
}

.section-title {
  font-size: var(--text-headline-md);
  font-weight: 700;
  color: var(--on-surface);
  letter-spacing: var(--tracking-headline-md);
  margin: 0 0 var(--space-lg);
}

.vendor-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-lg);
}

.vendor-card {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  transition: box-shadow 0.15s ease;
}

.vendor-card:hover {
  box-shadow: var(--shadow-2);
}

.vendor-icon {
  font-size: 28px;
  color: var(--secondary);
}

.vendor-card-title {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
  margin: 0;
}

.vendor-card-desc {
  margin: 0;
  font-size: var(--text-body-sm);
  color: var(--muted);
  line-height: 1.6;
}

.vendor-card-stat {
  font-size: var(--text-label-sm);
  color: var(--muted);
  margin-top: auto;
}

.vendor-stat-num {
  font-weight: 700;
  color: var(--on-surface);
  font-variant-numeric: tabular-nums;
}

@media (max-width: 768px) {
  .vendor-grid {
    grid-template-columns: 1fr;
  }
}
</style>
