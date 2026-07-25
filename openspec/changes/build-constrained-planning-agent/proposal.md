## Why

Plan generation currently treats the model as the primary planner and the backend as a validator or fallback. This creates long synchronous requests, allows the model to propose avoidable schedule conflicts, blocks useful local planning when AI is disabled or quota-limited, and labels deterministic output as AI-generated. The product needs its own planning Agent that understands account history, workload, dates, occupied time, and product rules while using the model only as a bounded planning collaborator.

## What Changes

- Make a backend planning Agent the authoritative orchestrator for validation, context collection, stage decomposition, scheduling, conflict repair, and final verification.
- Generate a deterministic conflict-free candidate plan before calling any model.
- Add local stage templates for learning, reading, exam preparation, project delivery, and general goals.
- Send the model a precise planning brief containing the normalized goal, local stage skeleton, candidate dates/times, workload profile, and safe aggregate history.
- Allow the model to enrich task semantics and suggest schedule adjustments, but never bypass Agent-owned constraints or persisted occupancy.
- Enforce one total interactive generation budget and return the valid local candidate when enrichment times out or fails.
- Separate local planning availability from model enablement and model quota.
- Return truthful source, enrichment status, warnings, fallback reason, and phase timing metadata.
- Add frontend loading, duplicate-submit prevention, source explanation, and conflict-adjustment warnings.

## Capabilities

### Modified Capabilities

- `ai-plan-generator`: Replace model-first generation with an Agent-orchestrated, constraint-first planning workflow and optional bounded model collaboration.
- `plan-management`: Reuse authoritative scheduling and overload rules while creating and committing generated previews.
- `admin-config`: Separate planning availability from model enrichment controls and configure the interactive enrichment time budget.

## Confirmed Decisions

- The Agent owns validation, dates, time ranges, workload, conflict repair, final plan totals, and persistence.
- SiliconFlow is an optional planning collaborator for semantic decomposition and enrichment, not the source of truth.
- Version one uses rule-based stage templates rather than a large maintained domain curriculum library.
- A valid local plan is returned when SiliconFlow is disabled, unavailable, over quota, invalid, or too slow.
- Model enrichment may improve titles, objectives, descriptions, summaries, rationale, and stage labels; schedule suggestions are accepted only after deterministic repair and revalidation.
- Existing `/api/ai/*` routes remain compatible during the client migration.
