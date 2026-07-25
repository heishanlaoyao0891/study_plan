## Context

The backend already has local date expansion, a rule fallback preview, learning profile queries, active-plan load, recent outcomes, persisted conflict checks, interval-union validation, manual schedule templates, and transactional commit. Recovery scheduling already demonstrates candidate search, preview, stale-state checks, and apply-time revalidation. These capabilities are fragmented across services and handlers, while the current generation handler calls the provider synchronously with a large token budget and up to two full attempts.

## Decisions

### Agent pipeline is deterministic before it is generative

Generation runs through explicit phases:

1. Validate and normalize goal, day count, start date, skip dates, desired daily hours, and available time range.
2. Load safe aggregate learning profile, active workload, outcomes, and exact occupied intervals.
3. Classify the goal into a local stage template and build progressive learning stages.
4. Allocate requested work into conflict-free candidate dates and time slots.
5. Validate the local candidate with the same schedule and workload rules used by normal plan creation.
6. Optionally ask SiliconFlow to enrich the fixed planning brief and suggest bounded adjustments.
7. Merge only allowed fields, repair schedule suggestions against current occupancy, and validate again.
8. Return the enriched candidate or the already valid local candidate with truthful metadata.

### Input contradictions become warnings or corrections

The Agent rejects malformed dates and impossible ranges. When desired daily hours exceed the supplied available interval, the interval is authoritative for that day and the response explains the reduced daily capacity or extends the plan duration when allowed. Skip dates and existing occupied intervals are always authoritative. The Agent searches alternate time within availability first, then a later eligible date, instead of returning a conflict that it can repair.

### Stage templates provide independent planning value

Version one includes small reusable templates:

- Learning: foundation, guided practice, independent practice, review.
- Reading: scope, progressive sections, notes, synthesis.
- Exam: diagnosis, topic cycles, timed practice, review and buffer.
- Project: requirements, setup, milestones, integration, review.
- General: understand, practice, apply, consolidate.

Templates determine stage order, task intent, review cadence, and buffer placement. They do not embed a large Web3, programming, language, or exam-specific curriculum.

### Model collaboration is bounded

The provider receives a typed, delimited brief and a canonical candidate skeleton. It cannot add cross-user data or raw records. Interactive generation advertises one 12-second overall request budget. Context queries, quota checks, validation, and enrichment all use the request context and share an 11.5-second work deadline, reserving the final 500 milliseconds for local response serialization. Retries are allowed only for transient failures and only inside that total deadline. Cancellation stops database and provider work; enrichment failures after a valid local candidate still return that candidate.

### Preview provenance is explicit

Responses distinguish `local` and `local_enriched`. Enrichment status distinguishes success, disabled, quota-limited, timeout, provider failure, and invalid output. Model quota counts only actual external enrichment attempts. Local planning does not consume model quota. Committed plans record generation source separately from the legacy `ai_generated` compatibility flag.

Each generated task receives an opaque identity. The signed, user-scoped, expiring provenance token binds plan title, summary, rationale, task count, and ordered task identities. The commit client does not submit a second purported original preview. The current review UI may edit task date, start/end time, title, objective, description, and difficulty; it may not alter plan metadata, task count, task identity, or task order. Estimated minutes, total hours, sorted persistence order, plan dates, and weekly target are recomputed server-side.

Commit idempotency is user-and-key scoped and payload-aware. SQLite busy and unique races receive bounded retries and winner reconciliation: the same key and normalized payload replays the winner, while the same key with a different normalized payload returns conflict.

### Compatibility precedes route renaming

Existing `/api/ai/generate-plan`, `/api/ai/regenerate`, and `/api/ai/commit-plan` remain available. Their response becomes typed and additive. A later client can migrate to neutral `/api/planning/previews` resources with preview ID, version, expiry, and idempotent commit.

## Risks / Trade-offs

- Local templates can feel generic -> use model enrichment when available and keep previews editable.
- Conflict-free allocation is more complex than rejecting conflicts -> reuse schedule resolution and recovery candidate search.
- A strict model deadline can reduce enrichment completion -> smaller semantic-only schema and no model-owned schedule generation reduce output size.
- Returning a local candidate after model failure can hide provider incidents -> expose bounded reason codes and phase timings to users/admins.
- Existing plans only have an AI boolean -> add source provenance without guessing historical details.

## Migration Plan

1. Add typed Agent request/response metadata and characterization tests behind existing routes.
2. Extract reusable schedule occupancy, validation, and candidate allocation services.
3. Replace fallback generation with the first-class local stage planner.
4. Change the model request to enrich the canonical skeleton under a total deadline.
5. Add source/provenance fields and preserve existing plan rows with an unknown legacy source.
6. Update the frontend from “AI generation” to “smart planning” with optional AI-enhanced status.
