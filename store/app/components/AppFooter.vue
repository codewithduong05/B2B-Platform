<template>
  <footer class="store-footer">
    <div class="footer-inner">
      <div class="footer-grid">
        <!-- Brand Column -->
        <div class="footer-brand">
          <div class="footer-brand-header">
            <span class="material-symbols-outlined footer-brand-icon">verified_user</span>
            <span class="footer-brand-text">Atlas SRM Assurance</span>
          </div>
          <p class="footer-brand-desc">
            Enterprise supplier relationship management, punchout catalog compliance, and multi-tier
            procurement orchestration adhering to strict ISO 9001:2015 and SOC 2 Type II audit
            standards.
          </p>
          <div class="footer-certs">
            <span class="footer-cert">ISO 9001 Certified</span>
            <span class="footer-cert">SOC 2 Type II</span>
            <span class="footer-cert">cXML PunchOut Ready</span>
          </div>
        </div>

        <!-- Sourcing Column -->
        <div class="footer-col">
          <h4 class="footer-col-title">Sourcing &amp; Vendors</h4>
          <nav class="footer-nav">
            <a href="/supplier-onboarding">Supplier Onboarding</a>
            <a href="/rfq">B2B Sourcing Events</a>
            <a href="/contracts">Master Service Agreements</a>
            <a href="/diversity">Diversity Sourcing</a>
          </nav>
        </div>

        <!-- Enterprise Terms Column -->
        <div class="footer-col">
          <h4 class="footer-col-title">Enterprise Terms</h4>
          <nav class="footer-nav">
            <a href="/terms">Commercial Terms of Sale</a>
            <a href="/credit">Revolving Credit Facilities</a>
            <a href="/compliance">Supply Chain Compliance</a>
            <a href="/sla">Enterprise SLA Guarantee</a>
          </nav>
        </div>

        <!-- System & Integration Column -->
        <div class="footer-col">
          <h4 class="footer-col-title">System &amp; Integration</h4>
          <div class="footer-system">
            <div class="footer-system-row">
              <span class="footer-status-dot"></span>
              <span class="footer-system-label">PunchOut API: Operational</span>
            </div>
            <p class="footer-system-detail">Average Response: 42ms</p>
            <p class="footer-system-detail">SLA Uptime: 99.98%</p>
            <a href="/docs/api" class="footer-system-link"
              >Integration Docs &amp; Webhooks &rarr;</a
            >
          </div>
        </div>
      </div>

      <!-- Bottom Bar -->
      <div class="footer-bottom">
        <p class="footer-copyright">
          &copy; 2025 Atlas SRM Technologies Inc. All enterprise rights reserved. Governed under
          Master Agreement #MA-77109.
        </p>
        <nav class="footer-legal">
          <a href="/privacy">Privacy Policy</a>
          <a href="/security">Security Architecture</a>
          <a href="/noc">Global NOC Status</a>
        </nav>
      </div>
    </div>
  </footer>
</template>

<style scoped>
.store-footer {
  width: 100%;
  background-color: var(--surface-container-low);
  margin-top: var(--space-2xl);
  box-shadow: 0 1px 8px rgba(0, 0, 0, 0.04);
}

.footer-inner {
  max-width: 1440px;
  margin: 0 auto;
  padding: var(--space-2xl) var(--margin);
}

.footer-grid {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: var(--space-xl);
}

/* ── Brand ── */
.footer-brand {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.footer-brand-header {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.footer-brand-icon {
  font-size: 24px;
  color: var(--secondary);
}

.footer-brand-text {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
}

.footer-brand-desc {
  font-size: var(--text-body-md);
  color: var(--muted);
  max-width: 360px;
  margin: 0;
  line-height: 1.6;
}

.footer-certs {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  margin-top: var(--space-xs);
}

.footer-cert {
  padding: var(--space-sm) var(--space-md);
  border-radius: var(--radius);
  background-color: var(--surface-container);
  font-size: var(--text-label-sm);
  color: var(--muted);
  font-weight: 500;
}

/* ── Columns ── */
.footer-col {
  display: flex;
  flex-direction: column;
}

.footer-col-title {
  font-size: var(--text-label-md);
  font-weight: 600;
  color: var(--on-surface);
  text-transform: uppercase;
  letter-spacing: var(--tracking-label);
  margin: 0 0 var(--space-md);
}

.footer-nav {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.footer-nav a {
  font-size: var(--text-body-md);
  color: var(--muted);
  text-decoration: none;
  transition: color 0.15s ease;
}

.footer-nav a:hover {
  color: var(--on-surface);
}

/* ── System ── */
.footer-system {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
  font-size: var(--text-body-md);
  color: var(--muted);
}

.footer-system-row {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.footer-status-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  background-color: var(--status-ok);
  flex-shrink: 0;
}

.footer-system-label {
  font-size: var(--text-label-md);
  font-weight: 500;
  color: var(--on-surface);
}

.footer-system-detail {
  margin: 0;
  font-size: var(--text-body-sm);
}

.footer-system-link {
  font-size: var(--text-label-md);
  font-weight: 500;
  color: var(--secondary);
  text-decoration: none;
  margin-top: var(--space-xs);
}

.footer-system-link:hover {
  color: var(--on-surface);
}

/* ── Bottom ── */
.footer-bottom {
  margin-top: var(--space-2xl);
  padding-top: var(--space-lg);
  border-top: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--space-md);
}

.footer-copyright {
  margin: 0;
  font-size: var(--text-body-sm);
  color: var(--muted);
}

.footer-legal {
  display: flex;
  align-items: center;
  gap: var(--space-lg);
}

.footer-legal a {
  font-size: var(--text-body-sm);
  color: var(--muted);
  text-decoration: none;
  transition: color 0.15s ease;
}

.footer-legal a:hover {
  color: var(--on-surface);
}

@media (max-width: 768px) {
  .footer-grid {
    grid-template-columns: 1fr;
  }
}
</style>
