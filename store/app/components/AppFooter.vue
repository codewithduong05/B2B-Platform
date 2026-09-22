<template>
  <footer class="store-footer">
    <div class="footer-inner">
      <div class="footer-grid">
        <!-- Brand Column -->
        <div class="footer-brand">
          <div class="footer-brand-header">
            <span class="material-symbols-outlined footer-brand-icon">verified_user</span>
            <span class="footer-brand-text">Atlas SRM</span>
          </div>
          <p class="footer-brand-desc">
            Enterprise supplier relationship management and multi-tier
            procurement orchestration.
          </p>
        </div>

        <!-- Quick Links Column -->
        <div class="footer-col">
          <h4 class="footer-col-title">Quick Links</h4>
          <nav class="footer-nav">
            <a href="/catalog">Catalog</a>
            <a href="/orders">Orders</a>
            <a href="/support">Support</a>
            <a href="/faq">FAQ</a>
          </nav>
        </div>

        <!-- System Status Column -->
        <div class="footer-col">
          <h4 class="footer-col-title">System Status</h4>
          <div class="footer-system">
            <div class="footer-system-row">
              <span class="footer-status-dot"></span>
              <span class="footer-system-label">Platform: Operational</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Bottom Bar -->
      <div class="footer-bottom">
        <p class="footer-copyright">
          &copy; 2025 Atlas SRM Technologies Inc. All rights reserved.
        </p>
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
