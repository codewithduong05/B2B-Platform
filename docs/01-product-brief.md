# 01 — Product Brief

**Status:** Blueprint
**Owner:** Product

## Purpose

Define what Atlas is, who uses it, what it does, and what it deliberately does not do. Everything downstream — architecture, data model, features — must trace back to this document.

---

## What Atlas is

Atlas is a **multi-supplier B2B wholesale ordering platform**. Verified businesses order goods from multiple independent distributors through a single storefront, on commercial terms negotiated per buyer.

Atlas is not a consumer shop. Buyers are companies with licences, credit terms, and repeat purchasing patterns. Suppliers are independent businesses with their own stock, pricing, and delivery capability. Atlas is the layer that makes ordering across all of them coherent: one basket, one checkout, correct pricing per buyer, correct stock per supplier, one order history.

### The problem

A mid-size kitchen today orders from eight to fifteen distributors. That means eight to fifteen phone calls, spreadsheets, WhatsApp threads, and price lists that go stale the moment they are sent. Ordering is slow, prices are opaque, and nobody can answer "what did we spend on this item last quarter" without manual reconciliation.

Atlas replaces that with one ordering surface, one price book per buyer, one order history, and one statement — while each supplier keeps its own business.

### Why this is hard

The difficulty is not the storefront. It is that the platform must be **authoritative about things it does not fully own**:

- **Stock** is owned by a supplier's ERP, not by Atlas. Atlas must reflect it accurately and handle the moments when it disagrees.
- **Price** depends on who is asking. Two buyers see different prices for the same item, and one buyer may see different prices per supplier.
- **What may be ordered** depends on the buyer's verification status and the product's handling requirements.
- **Delivery** is per supplier, so one basket becomes several shipments with different timings.

Every one of these is a correctness boundary, not a display concern.

---

## Actors

| Actor | Who they are | What they need |
|---|---|---|
| **Buyer** | A purchasing manager, head chef, or owner at a food-service business | Find items, see their own prices, order fast, track deliveries, check statements |
| **Buyer staff** | An employee ordering on the buyer's behalf | Same, with narrower permissions |
| **Supplier** | A distributor or producer selling on Atlas | Manage listings, stock, prices, and orders for their own catalogue only |
| **Operations staff** | Atlas employees running day-to-day | Monitor orders, resolve fulfilment problems, handle exceptions |
| **Verification reviewer** | Operations staff with a compliance remit | Review business verification applications, grant or revoke purchasing scope |
| **Catalogue manager** | Atlas employees curating the catalogue | Products, categories, content, merchandising |
| **Finance staff** | Atlas employees handling money | Invoices, payments, reconciliation, supplier settlement |
| **Platform admin** | Atlas employees administering the system | Users, roles, settings, feature flags, audit |
| **Partner** | An external referrer sending buyers to Atlas | Attribution, reporting on what they referred |
| **External ERP** | The system of record for stock and dispatch | Push stock levels, receive orders, report fulfilment |

---

## Capability scope

Nine capability areas. Each maps to backend modules in [04 Backend Architecture](04-backend-architecture.md) and to concrete screens in [08 Feature Inventory](08-feature-inventory.md).

### 1. Identity and access
Buyer registration, business verification, session management, addresses, and role-scoped permissions for both buyer and staff users.

### 2. Catalogue
Products with unit-of-measure conversions, categories, brands, suppliers, handling requirements (ambient / chilled / frozen), and searchable attributes.

### 3. Pricing
Per-buyer price lists, quantity break tiers, contract rates, and effective-date ranges. Price is computed server-side and never trusted from the client.

### 4. Inventory
Stock visibility per supplier, lot and expiry tracking, reservations, FEFO allocation, quarantine, and low-stock alerting.

### 5. Commerce
Cart with per-supplier grouping, checkout, order creation, order splitting, fulfilment, shipment tracking, invoices, and credit exposure.

### 6. Payments
Payment methods (account credit, bank transfer, card, cash on delivery), payment intents, refunds, and reconciliation against orders.

### 7. Promotions
Discounts, vouchers, volume deals, bundles, and campaign scheduling — with eligibility rules and reporting.

### 8. CRM and partners
Leads, referral codes, partner registry, lead assignment, consultation requests, and attribution reporting.

### 9. Content and platform
Articles, static pages, navigation menus, banners, site settings, SEO metadata, notifications, media, feature flags, and audit logging.

### 10. AI assistance
Optional, model-backed help for buyers and staff: turning prose, photos, and voice into proposed order lines; ranking substitutes; extracting fields from documents; drafting content and replies; flagging anomalies; and a natural-language path to reporting. Every output is a **proposal** a person confirms, and every capability has a working non-AI path. Boundaries are specified in [12 AI Features](12-ai-features.md).

---

## Domain rules

These hold across the whole product. They are not implementation details.

| # | Rule | Why it matters |
|---|---|---|
| R1 | **Price is computed server-side.** The client may display a price but never submits one. | A client-supplied price is a revenue-loss bug waiting to happen. |
| R2 | **Stock is never promised, only attempted.** Availability shown is informational; a reservation is only real once the backend confirms it. | The ERP is authoritative and can disagree at any moment. |
| R3 | **A buyer's purchasable catalogue is determined by verification status and product handling requirement.** | Unverified buyers must not reach restricted lines. |
| R4 | **One basket produces one order record per supplier.** | Fulfilment, invoicing, and settlement are per supplier. |
| R5 | **Allocation follows FEFO** — earliest expiry first — within a lot-tracked product. | Perishable stock must not be stranded. |
| R6 | **A lot that is recalled, expired, or quarantined can never be allocated.** | Non-negotiable safety boundary. |
| R7 | **Money movements are idempotent.** Repeating a payment callback must not double-apply. | Gateways retry. Networks fail mid-request. |
| R8 | **Every state change on an order, payment, verification, or stock item is auditable** with actor, timestamp, and before/after. | Disputes are resolved with evidence, not memory. |
| R9 | **Buyers see only their own data; suppliers see only their own catalogue and orders.** | Tenant isolation. |
| R10 | **Deleting is soft.** Records that financial or legal reporting depends on are never hard-deleted. | History must survive. |
| R11 | **AI proposes; deterministic code decides.** No model output sets a price, a stock level, an eligibility outcome, or a compliance decision. | A probabilistic system cannot meet a reproducibility or audit requirement. |
| R12 | **Every AI-assisted action carries provenance.** Prompt version, model, input hash, and reviewer are recorded with the action. | A decision that cannot be explained cannot be defended. |

---

## Explicit non-goals

Stated so nobody builds them by accident.

| Not doing | Reason |
|---|---|
| Consumer (B2C) retail | Different problem: no verification, no contract pricing, no credit terms |
| Being the system of record for stock | The ERP owns stock. Atlas reflects it. |
| Own delivery fleet or route optimisation | Suppliers deliver. Atlas tracks, it does not dispatch. |
| Marketplace payouts and escrow | Atlas bills the buyer directly; supplier settlement is an accounting process, not a payments product |
| Full accounting / general ledger | Atlas produces statements and reconciliation inputs. Accounting happens elsewhere. |
| Real-time chat between buyer and supplier | Out of scope for v1. Contact details and order notes only. |
| Native desktop applications | Web and mobile cover the need |
| Multi-currency in v1 | Single currency simplifies pricing, credit, and reconciliation. Revisit later. |
| Self-service supplier onboarding without review | Every supplier is reviewed before going live |
| Autonomous AI decisions on money, stock, eligibility, or compliance | Not reproducible, not auditable, not safe |
| AI-generated content published without human review | Puts unreviewed claims in front of customers |
| Free-form SQL generated by a model against the database | Unbounded access, unbounded cost, unauditable |
| Any AI feature without a non-AI path | Turns an optional enhancement into a hidden correctness dependency |

---

## Success measures

Design targets. Not measured yet — nothing has been built.

| Measure | Target | Why |
|---|---|---|
| Time to place a repeat order | Under 2 minutes for a 20-line basket | Speed is the entire value proposition |
| Order lines requiring manual intervention | Under 2% | Manual fixes do not scale |
| Price disputes per 100 orders | Under 0.5 | Server-side pricing must be visibly correct |
| Stock-accuracy complaints | Under 1% of orders | ERP drift must be surfaced, not hidden |
| Buyer verification turnaround | Under 1 business day | Blocks first order |
| Checkout success rate | Above 97% | Excludes user-abandoned carts |

---

## Assumptions and open questions

**Assumptions**
- Buyers are businesses with a verifiable trading identity.
- Suppliers deliver within an agreed service window.
- One currency, one tax jurisdiction, one language at launch.
- The ERP integration is eventually consistent, not real-time-locking.

**Open questions to resolve while building**
1. Should price lists support time-boxed promotional overrides, or should that stay entirely in the promotions module?
2. When the ERP reports stock lower than an already-confirmed reservation, does Atlas short-ship, substitute, or hold for confirmation?
3. Is buyer-level credit exposure enforced as a hard block at checkout, or as a warning with an approval step?
4. Do suppliers see the buyer's negotiated price, or only their own payout?
5. How long is verification evidence retained, and who can retrieve it?
6. Does a basket spanning suppliers with different delivery zones still produce a single checkout?

Record the answers as they are decided. Do not let them stay implicit in code.

---

## Related

- [02 System Architecture](02-system-architecture.md) — how these capabilities are structured
- [08 Feature Inventory](08-feature-inventory.md) — these capabilities as concrete screens and endpoints
- [11 Conventions & Glossary](11-conventions-and-glossary.md) — the vocabulary used above
