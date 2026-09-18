# 11 — Conventions & Glossary

**Status:** Blueprint
**Owner:** Engineering Lead

## Purpose

The rules everyone follows, and the shared vocabulary. Read before your first pull request. When this document and a code review disagree, one of them is wrong — reconcile it rather than letting both stand.

---

# Part 1 — Conventions

## Language

| Context | Language |
|---|---|
| Chat, verbal discussion | The team's working language |
| Code: identifiers, comments, commit messages | English |
| API error messages | English, technical, developer-facing |
| Code documentation (docstrings, JSDoc) | English |
| Architecture and engineering documents | English |
| Conventional commit summary line | English |
| User-facing copy | Through the localisation layer — never hardcoded |
| Requirements and specifications | The team's working language |

**Why.** Mixed-language identifiers make code ungreppable and reviews slow. User-facing copy must be localisable regardless of the author's language.

---

## Git

### Branches

```
main                  # always deployable
├── feature/<ticket>-short-description
├── fix/<ticket>-short-description
├── chore/<ticket>-short-description
├── docs/<ticket>-short-description
└── hotfix/<ticket>-short-description
```

**Rules**
1. `main` is protected. No direct pushes.
2. Branch from `main`, merge back into `main`.
3. Rebase before merging to keep history linear.
4. Delete the branch after merge.
5. A `hotfix/*` branch starts from the deployed tag, not from `main`, when `main` contains unreleased work.

### Commits

Conventional commits. One logical change per commit.

```
<type>(<scope>): <summary>

<body — what and why, not how>

<footer — references, breaking changes>
```

| Type | Use |
|---|---|
| `feat` | New behaviour |
| `fix` | Bug fix |
| `refactor` | No behaviour change |
| `perf` | Performance improvement |
| `test` | Tests only |
| `docs` | Documentation only |
| `chore` | Tooling, dependencies, configuration |
| `build` | Build system |
| `ci` | CI configuration |
| `revert` | Reverting a previous commit |

**Scope** is the module name: `feat(checkout): ...`, `fix(inventory): ...`.

**Rules**
- Summary in the imperative mood: "add", not "added" or "adds".
- Summary under 72 characters.
- The body explains **why**. The diff explains what.
- A breaking change is marked explicitly in the footer.
- Reference the ticket.

**Never** commit: secrets, credentials, `.env` files, build output, dependency directories, large binaries, editor configuration.

### Pull requests

| Rule | Detail |
|---|---|
| Small | One concern, under roughly 400 changed lines excluding generated code and fixtures |
| Described | What changed, why, how to verify, what is not covered |
| Tested | New behaviour has tests; a bug fix has a test that failed first |
| Reviewed | At least one approval from someone other than the author |
| Green | All CI checks pass |
| Self-reviewed | Read your own diff before requesting review |

**Author checklist**
- [ ] Does it do one thing?
- [ ] Do the tests prove the behaviour, not just exercise the code?
- [ ] Are failure paths handled?
- [ ] Checked for secrets and debug output?
- [ ] Are logs and metrics added?
- [ ] Does anything need updating in the documentation?
- [ ] Is there a rollback path?

---

## Code

### Naming

| Thing | Convention | Example |
|---|---|---|
| Python | `snake_case` | `checkout_service` |
| Python constants | `UPPER_SNAKE` | `MAX_PAGE_SIZE` |
| Python classes | `PascalCase` | `OrderService` |
| TypeScript variables | `camelCase` | `orderTotal` |
| TypeScript types | `PascalCase` | `OrderSummary` |
| Vue components | `PascalCase` files | `OrderSummaryCard.vue` |
| Composables | `use` prefix | `useCart` |
| Booleans | `is` / `has` / `can` prefix | `isEligible` |
| Database tables | singular `snake_case` | `order_line` |
| Domain events | `<entity>.<past_tense_verb>` | `order.placed` |

### Structure

| Rule | Detail |
|---|---|
| One concept per file | A file exporting two unrelated things should be two files |
| Functions have one job | If naming it needs "and", split it |
| Short functions | Aim under 40 lines; treat 80 as a hard limit |
| Shallow nesting | Aim under 3 levels; early returns over else-chains |
| Explicit dependencies | Inject, do not reach for a global |
| No magic numbers | Name every constant that carries meaning |
| Immutability by default | Mutate deliberately, not incidentally |

### Comments

Comments explain **why**, never what. The code says what it does.

| Comment this | Not this |
|---|---|
| Why FEFO rather than FIFO | `# loop over lots` |
| Why this timeout is 30s | `# set timeout` |
| Why this lock order matters | `# lock rows` |
| Why this apparently redundant check exists | `# check` |

Delete a comment that restates the code. A stale comment is worse than none.

### Error handling

| Rule | Detail |
|---|---|
| Never swallow | An empty `except` or `catch` is a defect |
| Catch specifically | Never catch everything unless you re-raise or handle meaningfully |
| Fail loudly | A silent failure becomes silent corruption |
| Add context | An error without context cannot be debugged |
| No control flow by exception | Expected outcomes are values; exceptions are for the unexpected |
| Never log secrets | Redact tokens, credentials, and personal data |

---

## Security

Non-negotiable.

| # | Rule |
|---|---|
| 1 | **Never commit a secret.** Not in code, tests, fixtures, examples, or documentation. |
| 2 | **Server-side authorisation only.** A client-side check is a UX affordance, never a control. |
| 3 | **Validate all input at the boundary.** Every parameter, every payload, every webhook. |
| 4 | **Never build SQL by string concatenation.** Parameterised queries only. |
| 5 | **Never log sensitive data.** Tokens, passwords, documents, and card data are redacted or omitted. |
| 6 | **Hash passwords with a modern algorithm.** Never reversible, never a plain digest. |
| 7 | **Opaque identifiers in public routes.** Sequential integers invite enumeration. |
| 8 | **Verify webhook signatures against the raw body.** Never a re-serialised one. |
| 9 | **Least privilege.** Every service, role, and credential has only what it needs. |
| 10 | **Dependencies are reviewed and pinned.** An unpinned dependency is an unpinned incident. |
| 11 | **Escaping by default in templates.** Never disable it without a written reason. |
| 12 | **Report a suspected vulnerability privately.** Never in a public tracker. |
| 13 | **Model output is untrusted input.** Validate against a schema before use. Never execute it, never interpolate it into SQL, never let it select a tool. |
| 14 | **Retrieved content is data, never instruction.** A supplier document, a buyer note, or a scraped page cannot change a prompt or trigger an action. |
| 15 | **No model provider SDK outside `modules/ai/`.** One gateway, so cost, redaction, and prompt versions stay answerable. |
| 16 | **Classify before prompting.** Buyer contract prices and personal data do not leave the boundary without explicit classification and approval. |

Rules 13–16 are explained in [12 AI Features](12-ai-features.md). They are security rules, not style preferences.

---

## Documents

### Required documents

| Document | Purpose | Owner |
|---|---|---|
| This set | Architecture and conventions | Architecture |
| Module README | Purpose, owned tables, events, public surface | Module owner |
| ADR | Why a significant decision was made | Decision maker |
| Runbook | How to handle a specific operational scenario | Platform |
| Screen specification | Every mobile screen, state, and edge case | Mobile |
| API reference | Generated from the OpenAPI document | Backend |

### Architecture decision records

Write an ADR when a decision is expensive to reverse, affects more than one module, or contradicts an existing document.

| Field | Content |
|---|---|
| Title | `ADR-0001: Use a modular monolith` |
| Status | Proposed / Accepted / Superseded |
| Context | The problem and constraints |
| Decision | What was decided |
| Consequences | What becomes easier, what becomes harder |
| Alternatives | What was considered and why it was rejected |

**Rules**
- ADRs are immutable. Supersede, never edit.
- An ADR is written when the decision is made, not when someone remembers.
- A decision that contradicts an ADR requires a new ADR.

### Document status

| Status | Meaning |
|---|---|
| **Blueprint** | A specification to build against. Nothing implemented. |
| **Draft** | Being written. May change substantially. |
| **Active** | Reflects the current system. Verified. |
| **Deprecated** | Still read, but superseded. Points to the replacement. |

**Rule:** a document that claims to describe a system must say whether it was **verified** against that system or merely **intended**. A design that was never checked against reality is a hypothesis, not a description.

---

# Part 2 — Glossary

| Term | Meaning |
|---|---|
| **Abstention** | A model declining to answer because grounding is missing. A correct outcome, not a failure. |
| **ADR** | Architecture Decision Record. See above. |
| **AI proposal** | A model-generated suggestion a human must confirm. The only shape AI output may take. |
| **Allocation** | Choosing which stock lots satisfy an order line. |
| **Alloy** | Grafana Alloy, the telemetry collector. Receives OTLP and is the only component that names a storage backend. |
| **API** | The FastAPI service exposing `/api/v1`. |
| **Admin** | The staff-facing Nuxt application. |
| **Audit log** | An append-only record of who changed what, when, and from what. |
| **Availability** | Whether an item can currently be ordered, based on stock and eligibility. |
| **BFF** | Backend for Frontend. The Nuxt server tier that holds the session and proxies the API. |
| **Beat** | The Celery scheduler. Enqueues periodic tasks; performs no work. |
| **Boundary** | A rule preventing one module from reaching into another's internals. |
| **Buyer** | A verified business purchasing through Atlas. |
| **Buyer staff** | A sub-account of a buyer with narrower permissions. |
| **Cardinality** | The number of distinct values a metric label takes. High cardinality breaks the metrics store, which is why identifiers never become labels. |
| **Cart** | An unsubmitted basket of lines, grouped by supplier. |
| **Checkout** | The act of submitting a cart, producing one order per supplier. |
| **Code** | An opaque public identifier used in buyer-facing routes. |
| **Confidence** | The reported certainty of a proposal. Must be surfaced in the UI, never hidden. |
| **Credit account** | A buyer's account terms: limit, ageing, exposure. |
| **Credit note** | A negative invoice for a return or adjustment. |
| **Cursor pagination** | Key-based paging for large or scrolling result sets. |
| **Dart** | The language a Flutter application is written in. |
| **Dead letter** | Where a task or event goes after exhausting retries. |
| **Domain event** | A past-tense fact published for other modules to consume. |
| **Drift** | Divergence between Atlas and the ERP. |
| **Eligibility** | What a buyer is permitted to see and order. |
| **ERP** | The external system of record for stock and dispatch. |
| **Feature flag** | A server-controlled switch that enables or disables behaviour without a deploy. On mobile it is the only practical rollback. |
| **FEFO** | First-Expired-First-Out allocation. |
| **Flutter** | A cross-platform application toolkit producing one codebase for Android and iOS. One of the three buyer applications in [09 Mobile Applications](09-mobile-application.md). |
| **Golden set** | A curated input/output set per AI feature, versioned with the prompt and used for scoring. |
| **Grafana** | The single pane over metrics, logs, and traces. Operational questions, not business ones. |
| **Grounding** | Constraining an answer to supplied source material, with citation. |
| **Guardrail** | A control on AI input or output: redaction, injection detection, schema validation, budget. |
| **Handling class** | The storage requirement governing a product: ambient, chilled, frozen. |
| **Human review queue** | Where sampled or low-confidence AI output waits for a person to accept, edit, or reject it. |
| **Idempotency key** | A client-supplied token making a retried mutation safe. |
| **Invoice** | A billing document for a fulfilled order. |
| **Jetpack Compose** | The Android native UI toolkit used by the Android buyer application. |
| **Lot** | A traceable batch of stock with its own expiry. |
| **Metabase** | The business analytics tool. Reads a read replica through curated models, never the primary database. |
| **Model gateway** | The single path through which every model call passes. Owned by the `ai` module. |
| **Modular monolith** | One deployable containing independently structured modules. |
| **Module** | A bounded area of the backend owning one schema and one public surface. |
| **OpenSearch** | The search index. Derived, never authoritative. |
| **OpenTelemetry** | The instrumentation standard the application depends on. It knows this interface, never a backend product. |
| **Order** | A commitment to supply, per supplier. |
| **Order line** | One product, quantity, and price on an order. |
| **OTLP** | The OpenTelemetry wire protocol. The one address the application needs to know. |
| **Partial success** | A multi-supplier checkout where some suppliers succeed and others do not. |
| **Plugin** | A packaged bridge from a cross-platform toolkit to a platform capability. Accepted only against the evaluation rule in [09 Mobile Applications](09-mobile-application.md). |
| **Price list** | A named set of prices assignable to buyers. |
| **Price tier** | A quantity break within a price list entry. |
| **Projection** | A derived local table built from another module's events. |
| **Promotion** | A time-boxed commercial offer. |
| **Prompt injection** | An attempt to make untrusted content alter a prompt or trigger an action. Treated as an attack, always. |
| **Prompt registry** | The versioned, owned store of prompts. A prompt without a version is not promoted. |
| **Proposal envelope** | The standard response shape for every AI route: proposal id, confidence, items, unresolved, provenance, expiry. |
| **Provenance** | The record of which prompt version, model, and input hash produced an output. |
| **Purchase scope** | The eligibility grant allowing a buyer to order restricted lines. |
| **Quarantine** | Marking a lot unavailable, usually for safety. |
| **Replica** | A read-only copy of the database kept in sync with the primary. Analytics reads this, never the primary. |
| **Reservation** | Stock held for an order not yet fulfilled. |
| **RBAC** | Role-based access control. |
| **RPO / RTO** | Recovery point / time objective. Data loss and downtime tolerances. |
| **Semantic cache** | A response cache keyed on input meaning rather than exact text. Used only where the tolerance is safe. |
| **Semantic layer** | The predefined metric and dimension definitions the analytics copilot resolves against. No free SQL. |
| **Shadow mode** | Running a new prompt or model alongside the current one, without serving its output. |
| **SLA** | Service level agreement. Used here for the internal fulfilment window. |
| **Soft delete** | Marking a record deleted without removing it. |
| **Span** | One timed unit of work inside a trace. A span is what turns a slow request into a diagnosis. |
| **Store** | The buyer-facing Nuxt application. |
| **Supplier** | An independent business selling through Atlas. |
| **Tenant** | An isolated customer of the platform: a buyer or a supplier. |
| **Token accounting** | Recording tokens and cost per call, attributed to a feature and a tenant. |
| **Trace** | The record of one request's path across services and tiers, made of spans and joined to logs by trace ID. |
| **SwiftUI** | The Apple native UI framework used by the iOS buyer application. |
| **Unit of measure** | How a product is counted. Conversions exist between them. |
| **Uncertain mutation** | A write whose outcome the client cannot determine. |
| **Verification** | Review confirming a business may buy on the platform. |
| **Vertical slice** | One complete flow built end to end, used to prove a stack rather than to ship a feature. |
| **Voucher** | A redeemable code applying a promotion. |
| **Webhook** | A signed inbound callback from an external system. |
| **Worker** | A Celery process consuming a queue. |

---

# Part 3 — Decision quick reference

Questions that come up repeatedly. Decide once, here.

| Question | Answer |
|---|---|
| New table? | In the owning module's schema. Never shared. |
| Need another module's data? | Public read, event subscription, or a projection. Never a join. |
| Slow operation? | Dispatch it. Return `202` with a status URL. |
| Need a new API version? | Only for a breaking change. Additive changes stay in place. |
| Where does authorisation live? | The API. Always. |
| Where does price come from? | The server. Always. |
| Optimistic update? | Yes for reversible UI. Never for money or commitment. |
| New admin route? | Add a route-registry entry with a permission, or it does not ship. |
| Retry a failed task? | Yes, with backoff, bounded. Then dead-letter and alert. |
| Retry a failed mutation? | Only with an idempotency key. Never automatically. |
| Delete this record? | Soft delete if finance or audit depends on it. |
| Add a dependency? | Justify it. Prefer the platform. Pin it. |
| Where does user-facing copy live? | The localisation layer. Never inline. |
| Cache an authenticated response? | Only with a tenant-scoped key. Test isolation. |
| Cross-schema foreign key? | No. Reference by value, verify in the service. |
| Where do business rules go? | The service layer. Not the router, not the client. |
| May AI set a price, stock level, or eligibility outcome? | No. Never. |
| May AI trigger a tool or an action on its own? | No. A deterministic decision in code does that. |
| Where does a model call go? | Through the `ai` module. A provider SDK outside it fails review. |
| Can I parse a completion with `json.loads`? | No. Validate against a Pydantic model or treat it as a failure. |
| Does this AI feature need a non-AI path? | Yes. Always. It is a release gate. |
| Can I change a prompt without an evaluation run? | No. A version without a score is not promoted. |
| Can I show a proposal as if it were confirmed? | No. Proposals are labelled, confirmed explicitly, and never auto-applied. |
| May a mobile client compute a price or a stock decision? | No. Every application displays what the server computed. |
| Can I share code between the three mobile apps? | Not required, and not the default. They coordinate through the API contract. |
| Must all three apps ship on the same day? | No. Each is an independent repository with its own release cycle. |
| Can I skip failure injection because the happy path passes? | No. A test suite that only proves the happy path proves nothing about a mobile client. |
| Where does the application send telemetry? | To the collector, through the OpenTelemetry API. Never directly to a storage backend. |
| A metric label with a buyer identifier on it? | No. Identifiers belong in traces and logs, where high cardinality is expected. |
| The collector is down. Does anything break? | No. Telemetry is a side channel. If it can become an outage, the instrumentation is wrong. |
| May a business user query the production database? | No. A read replica, through curated models, with a read-only account. |
| New reporting requirement. Where does it go? | Needed daily by staff doing their job: build it in the application. Exploratory or one-off: the analytics tool. |
| Can I put the analytics tool on the primary database? | No. Not for a read-only account, not for a small query, not temporarily. |

---

## Related

- [02 System Architecture](02-system-architecture.md) — the decisions behind these conventions
- [04 Backend Architecture](04-backend-architecture.md) — module-level rules
- [07 Frontend Architecture](07-frontend-architecture.md) — frontend rules
- [10 Delivery Plan](10-delivery-plan.md) — definition of done and review expectations
- [12 AI Features](12-ai-features.md) — the rules behind security items 13–16
