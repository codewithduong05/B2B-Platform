# Atlas Docs — Index

Documentation for the **Atlas B2B Platform** reference project. This is a fictional, standalone teaching project. See the [root README](../README.md) for what it is and why it exists.

---

## Reading order

### If you are building the project (primary path)

The build is organised into eight milestones, `M0` through `M7`. Read the full detail — deliverables, exit criteria, and risks — in the [10 Delivery Plan](10-delivery-plan.md) before starting each one.

If you are the person actually starting the work, read the [13 Kickoff Plan](13-kickoff-plan.md) first: it is the opening of M0 and M1 at day granularity, with the intern's and the owner's tasks separated.

| Milestone | Read first | Then |
|---|---|---|
| **M0** — Foundation | [01 Product Brief](01-product-brief.md) | [02 System Architecture](02-system-architecture.md), [03 Infrastructure](03-infrastructure.md), [14 Observability and Analytics](14-observability-and-analytics.md) |
| **M1** — Identity, catalogue, pricing | [04 Backend Architecture](04-backend-architecture.md) | [05 API Contract](05-api-contract.md), [08 Feature Inventory](08-feature-inventory.md) → Store §1–1.3 and Admin §2.2, §2.5 |
| **M2** — Inventory, commerce | [06 Data & Events](06-data-and-events.md) | [08 Feature Inventory](08-feature-inventory.md) → Store §1.4–1.5, Admin §2.3–2.4 |
| **M3** — Payments, operations | [05 API Contract](05-api-contract.md) → webhook rules | [08 Feature Inventory](08-feature-inventory.md) → Admin §2.7 |
| **M4** — Promotions, CRM, suppliers | [06 Data & Events](06-data-and-events.md) → events | [08 Feature Inventory](08-feature-inventory.md) → Admin §2.6, §2.8 |
| **M5** — Content, reporting, ERP | [03 Infrastructure](03-infrastructure.md) → deploy and runbooks | [08 Feature Inventory](08-feature-inventory.md) → Admin §2.9–2.11 |
| **M6** — Mobile | [09 Mobile Applications](09-mobile-application.md) | [08 Feature Inventory](08-feature-inventory.md) → Mobile section |
| **M7** — AI capabilities | [12 AI Features](12-ai-features.md) | [08 Feature Inventory](08-feature-inventory.md) → §1.7 and §2.12 |

**Days 1–10 of M0 and M1 are planned in detail** in [13 Kickoff Plan](13-kickoff-plan.md).

[07 Frontend Architecture](07-frontend-architecture.md) applies from **M1** onward — read it before writing your first page.

### If you need to understand one thing quickly

| Question | Document |
|---|---|
| What are we building, and for whom? | [01 Product Brief](01-product-brief.md) |
| How do the pieces connect? | [02 System Architecture](02-system-architecture.md) |
| What runs where, and how do I start it locally? | [03 Infrastructure](03-infrastructure.md) |
| How do I write a new backend module correctly? | [04 Backend Architecture](04-backend-architecture.md) |
| What does the API look like? | [05 API Contract](05-api-contract.md) |
| Where does data live, and what happens asynchronously? | [06 Data & Events](06-data-and-events.md) |
| How is the web frontend structured? | [07 Frontend Architecture](07-frontend-architecture.md) |
| **What features exist on each surface?** | [08 Feature Inventory](08-feature-inventory.md) |
| How are the mobile apps built? | [09 Mobile Applications](09-mobile-application.md) |
| What do I build, in what order? | [10 Delivery Plan](10-delivery-plan.md) |
| What do we call things, and how do we name code? | [11 Conventions & Glossary](11-conventions-and-glossary.md) |
| What is AI allowed to do here? | [12 AI Features](12-ai-features.md) |
| **What do I do in my first ten days?** | [13 Kickoff Plan](13-kickoff-plan.md) |
| How do I observe the system, and where do reports come from? | [14 Observability and Analytics](14-observability-and-analytics.md) |

---

## Document map

```
docs/
├── README.md                     # this file
├── 01-product-brief.md           # domain, actors, scope, non-goals
├── 02-system-architecture.md     # topology, layering, boundaries, decisions
├── 03-infrastructure.md          # local stack, deploy topology, CI/CD
├── 04-backend-architecture.md    # modular monolith, module anatomy, rules
├── 05-api-contract.md            # HTTP conventions, auth, endpoint catalogue
├── 06-data-and-events.md         # schemas, ERD, domain events, job queues
├── 07-frontend-architecture.md   # Store, Admin, shared design system, BFF
├── 08-feature-inventory.md       # feature list per surface (Store/Admin/API/Mobile)
├── 09-mobile-application.md      # three apps: Flutter, native Android, native iOS
├── 10-delivery-plan.md           # phased build plan, milestones, acceptance
├── 11-conventions-and-glossary.md  # language, git, code, security, docs, glossary
├── 12-ai-features.md             # AI boundaries, capabilities, guardrails, cost, evaluation
├── 13-kickoff-plan.md            # first ten days, intern tasks vs owner tasks
└── 14-observability-and-analytics.md  # OTel, Alloy, Grafana, and Metabase over a replica
```

---

## Conventions used in these documents

| Convention | Meaning |
|---|---|
| `code` (inline) | A module, file, field, route, or queue name |
| ```mermaid blocks | Diagrams — render them, they carry real information |
| Tables | Structured facts: ports, endpoints, features, decisions |
| **Bold** | A term being defined, or a rule that must not be relaxed |
| `atlas.internal` | A placeholder host. Never a real address. |

Requirements language:

- **MUST** — required. Breaking it is a defect.
- **SHOULD** — expected. Deviating needs a stated reason.
- **MAY** — optional.

---

## Important caveats

1. **No real infrastructure.** Every hostname, port, and credential in these documents is illustrative. Do not substitute a real one.
2. **No production data.** Seed data is generated for local development only.
3. **Design intent, not verified behaviour.** Nothing here has been built or measured. Performance figures are budgets to design against, not observed results.
4. **The domain is swappable.** Food-service wholesale is a vehicle for the architecture. If your exercise uses a different domain, keep the architecture and replace the vocabulary.
