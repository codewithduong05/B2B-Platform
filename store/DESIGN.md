# Atlas B2B SRM Platform — Design System

**Design System Name:** Enterprise Operational Precision
**Stitch Project:** [4437078058782272259](https://stitch.withgoogle.com/projects/4437078058782272259)
**Token Source:** `app/assets/styles/tokens.css`
**Last Synced:** 2026-09-21

---

## Brand & Style

This design system establishes a high-assurance, precision-engineered visual language for enterprise Supplier Relationship Management (SRM) and mission-critical procurement workflows. It bridges two interconnected environments:

1. **Buyer Storefront** — intuitive, high-trust interface for requisitioning and catalog discovery
2. **Operational Console** — information-dense dashboard for procurement officers, supply chain controllers, and enterprise finance teams

The design movement is **Corporate Modern with High-Density Utilitarianism**. The aesthetic eliminates decorative ornamentation, consumer SaaS gloss, and transient trends. Visual authority is achieved through meticulous structural alignment, strict typographic discipline, explicit tabular numeric rendering, and disciplined 1px grid divisions.

---

## Colors

### Core Brand Tokens

| Token | Hex | Usage |
|-------|-----|-------|
| Primary (Black) | `#000000` | Primary buttons, headers, structural ink |
| Secondary (Blue) | `#0051d5` | Interactive links, active states, focus rings |
| Tertiary (Purple) | `#7c3aed` | Financial/quotes, contract lifecycle |
| Background | `#f8f9ff` | Base application canvas |
| Surface | `#ffffff` | Working surfaces, cards, data grids |

### Surface Elevation System

| Level | Token | Hex | Usage |
|-------|-------|-----|-------|
| 0 | `--surface-container-lowest` | `#ffffff` | Pure white surfaces |
| 1 | `--surface-container-low` | `#eff4ff` | Secondary panels |
| 2 | `--surface-container` | `#e5eeff` | Elevated containers |
| 3 | `--surface-container-high` | `#dce9ff` | High-elevation panels |
| 4 | `--surface-container-highest` | `#d3e4fe` | Maximum elevation |

### Status Badge Hierarchy

Tripartite structure: solid text + 15% tint background + 30% border stroke.

| Status | Text | Background | Border |
|--------|------|------------|--------|
| Active/Confirmed | `#059669` | `#ecfdf5` | `#a7f3d0` |
| Pending/Processing | `#2563eb` | `#eff6ff` | `#bfdbfe` |
| Warning/Low Stock | `#d97706` | `#fffbeb` | `#fde68a` |
| Critical/Failed | `#dc2626` | `#fef2f2` | `#fecaca` |
| Archived | `#475569` | `#f8fafc` | `#e2e8f0` |

---

## Typography

**Font Family:** Inter (system-ui fallback)

### OpenType Features

All financial values, stock counts, SKU identifiers, and order sizes must enable:

```css
font-feature-settings: "tnum" 1, "cv05" 1, "cv11" 1;
```

This ensures vertical alignment down dense ledger rows and immediate visual verification of figures.

### Type Scale

| Level | Size | Weight | Line Height | Letter Spacing | Usage |
|-------|------|--------|-------------|----------------|-------|
| `display-lg` | 32px | 600 | 40px | -0.02em | Storefront display headlines |
| `headline-lg` | 24px | 600 | 32px | -0.015em | Section headers |
| `headline-md` | 20px | 600 | 28px | -0.01em | Card titles, admin headings |
| `headline-sm` | 16px | 600 | 24px | -0.005em | Sub-section headers |
| `body-lg` | 16px | 400 | 24px | 0em | Primary body text |
| `body-md` | 14px | 400 | 20px | 0em | Default body, data tables |
| `body-sm` | 12px | 400 | 16px | 0em | Compact table cells |
| `label-md` | 13px | 500 | 18px | 0.01em | Form labels |
| `label-sm` | 11px | 600 | 14px | 0.04em | Status badges, uppercase metadata |
| `tabular-lg` | 20px | 600 | 24px | -0.01em | KPI values |
| `tabular-md` | 14px | 500 | 20px | 0em | Financial table cells |
| `tabular-sm` | 12px | 500 | 16px | 0em | Compact financial data |

---

## Layout & Spacing

### Buyer Storefront

- **Max Width:** 1440px centered
- **Grid:** 12-column responsive
- **Margins:** 24px (`--margin`)
- **Gutters:** 16px (`--gutter`)

### Operational Console (Admin)

- **Layout:** Persistent left-rail (56px icon-only or 240px expanded)
- **Header:** 48px global action bar
- **Canvas:** Edge-to-edge with 16px (`--margin-admin`) margins
- **Grid Density:** 8px (`--gutter-dense`) or 16px (`--gutter`)

### Responsive Breakpoints

| Viewport | Behavior |
|----------|----------|
| Desktop (>1280px) | Full multi-pane: sidebar + data list + inspector |
| Tablet (768-1279px) | Inspector collapses to modal, left-rail to 56px |
| Mobile (<767px) | Single-column cards, bottom sheet navigation |

### Spacing Scale

| Token | Value | Pixels |
|-------|-------|--------|
| `--space-xs` | 0.25rem | 4px |
| `--space-sm` | 0.5rem | 8px |
| `--space-md` | 0.75rem | 12px |
| `--space-lg` | 1rem | 16px |
| `--space-xl` | 1.5rem | 24px |
| `--space-2xl` | 2rem | 32px |

---

## Elevation & Depth

Depth is achieved via **low-contrast architectural outlines** backed by micro-level structural elevations. No colorful ambient glows or blurred glass elements.

| Level | Shadow | Border | Usage |
|-------|--------|--------|-------|
| 0 | none | `1px solid #E2E8F0` | Inline tables, secondary panels |
| 1 | `0 1px 2px rgba(15,23,42,0.05)` | `1px solid #CBD5E1` | Resting cards, metric widgets |
| 2 | `0 4px 6px -1px rgba(15,23,42,0.08)` | `1px solid #CBD5E1` | Dropdowns, popovers |
| 3 | `0 20px 25px -5px rgba(15,23,42,0.12)` | `1px solid #CBD5E1` | Modals, slide-over drawers |

---

## Shapes

Disciplined, non-circular geometry centered on **ROUND_FOUR** (4px base).

| Radius | Value | Usage |
|--------|-------|-------|
| `--radius-sm` | 2px | Checkboxes, radio controls |
| `--radius` | 4px | Buttons, inputs, badges, dropdowns |
| `--radius-md` | 6px | Cards, containers, modals |
| `--radius-lg` | 8px | Large containers |
| `--radius-xl` | 12px | Special cases |

**Prohibited:** Pill buttons (fully rounded), floating capsules, asymmetrical organic shapes.

---

## Components

### Buttons

| Variant | BG | Text | Border | Height |
|---------|-----|------|--------|--------|
| Primary | `#0F172A` | `#FFFFFF` | transparent | 40px (store) / 32px (admin) |
| Secondary | `#FFFFFF` | `#334155` | `1px solid #CBD5E1` | 40px / 32px |
| Destructive | `#FFFFFF` | `#DC2626` | `1px solid #FECACA` | 40px / 32px |
| Ghost | transparent | `#64748B` | none | 40px / 32px |

### Data Grid Tables

- **Header:** BG `#F8FAFC`, height 36px, `label-sm`, color `#475569`
- **Rows:** Default height 44px (compact 36px), border-bottom `1px solid #E2E8F0`
- **Hover:** `#F8FAFC`
- **Selected:** `#EFF6FF`, left border `2px solid #2563EB`
- **Numeric Cells:** Right-aligned with `tabular-numeric-md`

### Input Fields

- **Height:** 40px (store) / 36px (admin)
- **Border:** `1px solid #CBD5E1`
- **Focus:** Border `#2563EB`, shadow `0 0 0 1px #2563EB`
- **Invalid:** Border `#DC2626`, shadow `0 0 0 1px #DC2626`

### Status Badges

- **Font:** `label-sm` (11px), uppercase, `0.04em` tracking
- **Radius:** 4px
- **Padding:** 2px 8px
- **Border:** 1px solid

---

## Screens (Stitch) — Full Analysis

### Screen Overview

| # | Screen | Stitch ID | Size | Route | Status |
|---|--------|-----------|------|-------|--------|
| 1 | Store Home & Procurement Dashboard | `27a5a4f8...` | 2560x4626 | `/` | ✅ Implemented |
| 2 | Catalogue - Product Listing & Specifications | `d6f39ba4...` | 2560x4388 | `/catalog` | ✅ Implemented |
| 3 | Search - Faceted Discovery & Exact SKU Matching | `ce6f6a69...` | 2560x3190 | `/search` | ❌ Missing |
| 4 | Product Detail: Low Moisture Mozzarella Loaf | `9f4d7c9a...` | 2560x4700 | `/product/:sku` | ✅ Implemented |
| 5 | Cart & Requisition Review | `5d93049e...` | 2560x4900 | `/cart` | ❌ Missing |
| 6 | Checkout & Purchase Submission | `86e2ad88...` | 2560x5130 | `/checkout` | ❌ Missing |
| 7 | Order Confirmation: ATL-2026-004821 | `2553c308...` | 2560x3458 | `/orders/:code` | ❌ Missing |
| 8 | Design System & Primitives | `6d2c253f...` | 2560x7582 | — | ✅ tokens.css |

---

### Screen 1: Store Home & Procurement Dashboard

**Route:** `/` | **Stitch:** `27a5a4f85f724714a436136cc9fe25b3` | **Status:** ✅

**Stitch Design Contains:**
- Top navigation bar with Atlas branding, search, Quick Order, RFQ, Cart icons
- Hero section with 4 KPI cards (Products, Categories, Brands, Suppliers)
- Quick-Order SKU panel with preset chips and quantity stepper
- Multi-vendor feature cards (Split Routing, Unified Billing, cXML/PunchOut)
- Contract compliance banner (MSA #MA-77109)
- Category browsing grid with icons and product counts
- Instant Replenishment tray (recent orders)
- Compliance badges strip (ISO 9001, REACH/RoHS, Lot Traceability, SOC 2)

**Codebase Implementation:**
- `pages/index.vue` → renders 6 child components
- `HeroSection.vue` → KPIs from `GET /api/home/stats`, quick-order via `useCart`
- `MultiVendorSection.vue` → supplier count from stats API
- `ContractBanner.vue` → static content
- `CategoryBento.vue` → categories from `GET /api/home/categories`
- `ReorderTray.vue` → empty state, wired to `useCart`
- `ComplianceStrip.vue` → static badges

**Gap:** None — fully implemented with real data.

---

### Screen 2: Catalogue - Product Listing & Specifications

**Route:** `/catalog` | **Stitch:** `d6f39ba4838a48bfa411ec9d41472fd3` | **Status:** ✅

**Stitch Design Contains:**
- Breadcrumb navigation
- Left filter sidebar: Supplier, Lead Time, Warehouse, MOQ slider, Compliance tags
- Top control bar: active filter pills, sort dropdown, grid/table toggle
- Product grid (3-column) with cards showing: supplier badge, SKU, image, name, description, stock, price, qty stepper, "Requisition" CTA
- Alternative: data table view with columns
- Pagination with page numbers and jump input

**Codebase Implementation:**
- `pages/catalog.vue` → orchestrates all child components
- `FilterSidebar.vue` → fetches suppliers from `GET /api/catalog/filters`
- `ControlBar.vue` → sort/view toggle, filter pills
- `ProductGrid.vue` → 3-column responsive grid
- `ProductCard.vue` → card with supplier badge, SKU, price, stock, qty stepper
- `ProductTable.vue` → data table alternative
- `Pagination.vue` → full pagination with jump input
- `useCatalog.ts` → `GET /api/catalog` with query params

**Gap:** None — fully implemented with real data.

---

### Screen 3: Search - Faceted Discovery & Exact SKU Matching

**Route:** `/search` | **Stitch:** `ce6f6a69c0fd4509bb586529b7aaa189` | **Status:** ❌ MISSING

**Stitch Design Contains:**
- Global search bar with autocomplete
- Search results with product cards
- Faceted filters (category, supplier, price range)
- SKU exact match highlighting
- Recent searches history
- Suggested products

**Codebase:** Not implemented. No `/search` page exists.

**Backend APIs Available:**
- `GET /api/v1/catalog/products?q=search_term` (text search)
- `GET /api/v1/catalog/products?sku=exact_sku` (SKU lookup)

**Components Needed:**
- `pages/search.vue`
- `SearchBar.vue` (global, in header)
- `SearchResults.vue`
- `SearchFacets.vue`
- `useSearch.ts` composable
- BFF: `GET /api/search?q=...`

---

### Screen 4: Product Detail

**Route:** `/product/:sku` | **Stitch:** `9f4d7c9aed734c7ebcebf80299b98795` | **Status:** ✅

**Stitch Design Contains:**
- Category breadcrumb with certifications/UNSPSC badges
- Image gallery with thumbnails, zoom, 3D CAD, diagram controls
- Engineering artifacts (CAD, datasheets) with download buttons
- Technical specs grid (key-value pairs)
- Purchase card: supplier badge, MPN, name, SKU
- Volume tier pricing table with active tier highlight
- Quantity stepper with +10/+50/+100 presets
- Stock radar (multi-warehouse availability)
- "Add to Staging Cart" + "Request Formal Volume Quote (RFQ)" CTAs
- Supplier profile card (vendor ID, tier, SLA, compliance)

**Codebase Implementation:**
- `pages/product/[sku].vue` → orchestrates all child components
- `ImageGallery.vue` → main viewport + thumbnails + badges
- `PriceCalculation.vue` → tier table + qty stepper
- `StockRadar.vue` → multi-warehouse stock
- `SpecsGrid.vue` → key-value spec rows
- `SupplierCard.vue` → vendor info + SLA metrics
- `EngineeringArtifacts.vue` → download cards
- `useProduct.ts` → `GET /api/product/:sku`
- `useCart.ts` → `POST /api/cart/items` (add to cart)

**Gap:** None — fully implemented with real data.

---

### Screen 5: Cart & Requisition Review

**Route:** `/cart` | **Stitch:** `5d93049e9aef4332ba8f9635326fb237` | **Status:** ❌ MISSING

**Stitch Design Contains:**
- Cart header with item count
- Line items: SKU, product name, supplier, unit price, quantity stepper, line total, remove button
- Cart summary: subtotal, estimated tax, shipping, total
- "Proceed to Checkout" CTA
- "Request Quote for Entire Cart" secondary CTA
- Empty cart state with "Browse Catalog" link
- Save for Later / Move to Requisition

**Codebase:** BFF routes exist (`GET /api/cart`, `POST/PATCH/DELETE /api/cart/items`), composable exists (`useCart.ts`), but NO page renders the cart.

**Components Needed:**
- `pages/cart.vue`
- `CartItem.vue` (line item row)
- `CartSummary.vue` (totals sidebar)
- `CartEmpty.vue` (empty state)

---

### Screen 6: Checkout & Purchase Submission

**Route:** `/checkout` | **Stitch:** `86e2ad8870d740e98d9ae1d268700664` | **Status:** ❌ MISSING

**Stitch Design Contains:**
- Checkout step indicator (Shipping → Review → Submit)
- Shipping address form / selection
- Delivery date selector
- Payment terms display (Net 30, etc.)
- Purchase order reference input
- Internal notes / cost center
- Order summary sidebar (items, subtotal, tax, total)
- "Submit Purchase Order" primary CTA
- "Save as Draft" secondary CTA

**Codebase:** No checkout page. Backend has `POST /api/v1/commerce/orders` (not yet proxied via BFF).

**Components Needed:**
- `pages/checkout.vue`
- `CheckoutShipping.vue`
- `CheckoutReview.vue`
- `CheckoutSummary.vue`
- BFF: `POST /api/checkout/submit`, `GET /api/checkout/summary`

---

### Screen 7: Order Confirmation

**Route:** `/orders/:code` | **Stitch:** `2553c308f9754bf2be9eca2292645d46` | **Status:** ❌ MISSING

**Stitch Design Contains:**
- Order confirmation header with order number (ATL-2026-004821)
- Order status timeline (Submitted → Confirmed → Processing → Shipped)
- Order details: items, quantities, unit prices, totals
- Shipping information
- Billing / payment terms
- Download invoice / receipt CTA
- "Track Shipment" CTA
- "Reorder" CTA

**Codebase:** No order pages. Backend has order APIs (not yet proxied).

**Components Needed:**
- `pages/orders/[code].vue`
- `OrderTimeline.vue`
- `OrderDetails.vue`
- BFF: `GET /api/orders/:code`

---

### Screen 8: Design System & Primitives

**Stitch:** `6d2c253f12ed418eb9f8b1587d93a974` | **Status:** ✅

**Stitch Design Contains:**
- Color palette swatches (all Material 3 tokens)
- Typography scale specimens
- Button variants (Primary, Secondary, Ghost, Destructive)
- Status badge specimens
- Card/container specimens
- Input field specimens
- Data grid table specimens
- Spacing and elevation specimens

**Codebase:** Implemented as `tokens.css` (447 lines) + `DESIGN.md` (this file).

**Gap:** None — tokens synced with Stitch design system.

---

## User Flow Map

```
┌─────────────────────────────────────────────────────────────────────┐
│                         BUYER STOREFRONT                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐     │
│  │  HOME    │───>│ CATALOG  │───>│ PRODUCT  │───>│   CART   │     │
│  │    /     │    │ /catalog │    │/product/ │    │   /cart  │     │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘     │
│       │               │               │               │             │
│       │               │               │               v             │
│       │               v               │         ┌──────────┐       │
│       │         ┌──────────┐          │         │ CHECKOUT │       │
│       │         │  SEARCH  │          │         │/checkout │       │
│       │         │ /search  │          │         └──────────┘       │
│       │         └──────────┘          │               │             │
│       │                               │               v             │
│       │                               │         ┌──────────┐       │
│       └───────────────────────────────┴────────>│  ORDER   │       │
│                                                 │ /orders/ │       │
│                                                 └──────────┘       │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Gap Summary

| Flow | Stitch | Codebase | BFF | Backend API | Priority |
|------|--------|----------|-----|-------------|----------|
| Home | ✅ | ✅ | ✅ | ✅ | — |
| Catalog | ✅ | ✅ | ✅ | ✅ | — |
| Product | ✅ | ✅ | ✅ | ✅ | — |
| **Search** | ✅ | ❌ | ❌ | ✅ | HIGH |
| **Cart** | ✅ | ❌ | ✅ | ✅ | HIGH |
| **Checkout** | ✅ | ❌ | ❌ | ⚠️ Partial | HIGH |
| **Order Confirmation** | ✅ | ❌ | ❌ | ⚠️ Partial | MEDIUM |

**Backend API Status:**
- ✅ Catalog, Product, Pricing, Inventory, Cart — fully working
- ⚠️ Orders, Checkout — endpoints exist but not all proxied via BFF
- ❌ Auth — `verifySessionToken` returns null (blocks cart/checkout/order flows)

---

## Design Token Mapping (Stitch → CSS)

| Stitch Token | CSS Variable |
|-------------|--------------|
| `primary` | `--accent` |
| `secondary` | `--secondary` |
| `tertiary` | `--tertiary` |
| `surface` | `--surface` |
| `background` | `--bg` |
| `on-surface` | `--on-surface` |
| `outline` | `--outline` |
| `surface-tint` | `--surface-tint` |
| `primary-container` | `--primary-container` |
| `secondary-container` | `--secondary-container` |

---

## References

- **Stitch Project:** https://stitch.withgoogle.com/projects/4437078058782272259
- **Tokens CSS:** `app/assets/styles/tokens.css`
- **Components:** `app/components/`
- **Pages:** `app/pages/`
