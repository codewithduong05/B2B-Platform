# API Coverage Audit — Executive Summary

**Coverage: 46/165 contract endpoints IMPLEMENTED (27.9%) · 3 PARTIAL · 116 MISSING · ~43 EXTRA · 0 LEGACY.** The Go backend fully covers cart → checkout → orders, inventory reservations, pricing, catalog, payments, and platform health. Entirely absent: identity routes (service exists, router is an empty stub), promotions, CRM, CMS, suppliers, AI (29 endpoints), ERP, platform surfaces, and inventory admin reads. Test evidence exists only for commerce, inventory, and payments — catalog, identity, and pricing have zero test files.

*Contract source: `atlas-platform/docs/05-api-contract.md`. Implementation source of truth: the Go source (`atlas-platform/backend/internal/modules/`). Status rule: IMPLEMENTED requires an actually registered route plus a working service path; handler files alone do not count; insufficient evidence is marked UNKNOWN (none needed — all rows below were traced to registered routes or their confirmed absence).*

## Endpoint Matrix

### identity — 0 IMP / 0 PART / 22 MISS. Router is a 22-line stub (`RegisterRoutes` = TODO comment); service layer (Register/Login/Refresh/Logout/passwords) exists but unreachable.
| Method | Contract path | Go route | Status | Task |
|---|---|---|---|---|
| all 22 (`/auth/*`, `/buyer/me/*`, `/admin/users`, `/admin/verifications`, `/admin/roles`) | — | *no routes registered* | MISSING (service exists, unmounted) | M1 leftover, no task |

### catalog — 14 IMP / 1 PART / 3 MISS. No test files.
| Contract path | Go route | Status |
|---|---|---|
| GET `/catalog/products`, `/products/{slug}`, `/categories`, `/brands`, `/suppliers`, `/facets` | same | IMPLEMENTED ×6 |
| GET `/catalog/search` | `GET /catalog/products/search` | PARTIAL (wrong path) |
| GET `/catalog/buyer-products` | — | MISSING |
| Admin products (list/create/update/publish/archive) | same paths | IMPLEMENTED ×5 |
| Admin categories list/create, suppliers list | same | IMPLEMENTED ×3 |
| Admin reindex ×2 | — | MISSING ×2 |
| EXTRA (not in contract): `/categories/{slug}`, `/brands/{slug}`, `/units`, `/handling-classes`, admin categories get/update/delete, admin brands ×5, admin units ×5, admin suppliers create/get/update/delete | — | EXTRA ×17 |

### pricing — 6 IMP / 0 PART / 0 MISS. No test files.
Contract 6/6 registered (`/admin/pricing/price-lists*`, `/pricing/quote`). EXTRA: `PATCH`/`DELETE /price-lists/{id}`, `GET …/assignments`.

### inventory — 1 IMP / 1 PART / 6 MISS. Tests ✓ (router + service).
| Contract path | Go route | Status |
|---|---|---|
| GET `/inventory/availability` | same | IMPLEMENTED |
| POST `/admin/inventory/lots/{id}/quarantine` | `POST /inventory/lots/{id}/quarantine` (buyer path, not `/admin/`) | PARTIAL (wrong prefix) |
| Admin stock/lots/release/low-stock/expiring/adjust | — | MISSING ×6 |
| EXTRA: `POST /reservations`, `GET /reservations/{id}`, `/request/{request_id}` (in `docs/inventory-api.md`, not 05) | — | EXTRA ×3 |

### commerce — 17 IMP / 0 PART / 2 MISS. Tests ✓ (7 files).
All cart (5), checkout, orders/me (2), admin orders list/detail/notes/hold/release (5), shipments list/update (2), invoices list/issue (2) IMPLEMENTED. MISSING: buyer cancel (deferred by decision), admin split (unscoped). EXTRA ×16: transition, shipments-create, invoice create/void/reissue, buyer returns, invoices/me ×2, returns admin ×4, credit-notes ×3, aged-debt.

### payments — 6 IMP / 1 PART / 0 MISS. Tests ✓ (3 files).
Methods, intents POST/GET, admin list, reconciliation, webhooks IMPLEMENTED. Refund PARTIAL (contract: single `POST /admin/payments/{id}/refund`; code: request/approve/reject trio). EXTRA: statements ×2, credit ×2.

### platform — 2 IMP / 9 MISS. `GET /health`, `/readyz` IMPLEMENTED (no tests); notifications/media/reference/flags/audit/traffic MISSING.
### promotions (7), CRM (9), CMS (14), suppliers (10), AI (29), ERP (5) — all MISSING. No modules, no routes, no tables.

## Totals
| Metric | Count |
|---|---|
| Contract endpoints | 165 |
| IMPLEMENTED | 46 (27.9%) |
| PARTIAL | 3 |
| MISSING | 116 |
| EXTRA (registered, not in contract) | ~43 |
| LEGACY | 0 |

## Findings
1. **Largest MISSING: AI module (29 endpoints)**, followed by CMS (14), platform surfaces (9), CRM (9).
2. **Largest PARTIAL: three-way tie (1 each)** — catalog search path, inventory quarantine prefix, payments refund shape.
3. **TASK correspondence for missing groups:** AI → M7 (no task ever opened); ERP/platform-notifications → M5 (no task); CRM/CMS/suppliers/promotions → M4 (no task); identity routes → M1 remainder (no dedicated task); catalog buyer-products/reindex + inventory admin reads → follow-ups never tasked; commerce split/cancel → explicitly deferred decisions.
4. **Is TASK-009 the correct next task?** Repository evidence alone cannot confirm a TASK-009 definition (none exists in `.ai/tasks.md`). By delivery-plan order, the next unbuilt vertical is **M4 (promotions/CRM/suppliers)** — promotions first is the plan-coherent choice since checkout already carries a voucher hook placeholder and M3's exit criteria are met. AI/ERP are explicitly later milestones (M7/M5) and should not lead. Smaller alternative: close M2 leftovers (admin split) or mount the identity router — both are narrower than a full M4 slice.
