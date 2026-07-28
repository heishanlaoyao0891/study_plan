## Why

The backend planning Agent already produces a fast, conflict-free local candidate, but the configured model is currently limited to an eight-second synchronous enrichment window and may only rewrite semantic fields on a fixed task skeleton. Even when the model succeeds, it cannot choose meaningful learning stages, vary the task count, or genuinely decompose the user's goal. In practice the recommended SiliconFlow model often exceeds the short interactive deadline, so users repeatedly receive a generic local plan and the external AI provides little product value.

The product needs AI to participate in curriculum and task decomposition without giving it authority over unsafe dates, occupied time, workload limits, or persistence. Model latency must be separated from the interactive HTTP response so H5 and mini-program users receive immediate feedback while the model receives enough time to produce useful work.

## What Changes

- Return a valid local baseline preview immediately and create a persistent asynchronous planning job when model decomposition is enabled.
- Ask the model for a compact learning blueprint containing stages, variable task count, objectives, difficulty, estimated effort, ordering, and prerequisite hints rather than a fully scheduled plan.
- Let the backend planning Agent convert the blueprint into concrete dates and time slots, repair capacity or occupancy conflicts, recompute totals, and perform final validation.
- Keep the local baseline available as a separately identified preview, but never publish or report it as a successful AI result when decomposition fails.
- Replace one-shot decomposition with a durable Agent loop that outlines, expands in bounded batches, normalizes safe deviations, validates, issues precise repair prompts, checkpoints progress, and resumes after restart.
- Support plans up to 30 days by constraining total effort to requested capacity and avoiding one oversized JSON response.
- Charge the user's daily AI generation allowance exactly once, only after a valid `ai_decomposed` preview is published; record provider attempts separately for operations and cost control.
- Accumulate bounded failure signatures and proven repair guidance in a versioned Prompt Playbook so recurring truncation and schema defects are prevented proactively.
- Use a versioned backend-owned blueprint prompt template that always supplies the complete normalized request, exact output schema, read-only scheduling constraints, and effective capacity derived from both desired hours and the actual available time slot.
- Make decomposition horizon-aware: short plans compress detail to cover the complete learning arc, while longer plans use finer tasks and pass bounded prior-stage/task progress into later batches; preserve every accepted model task through scheduling rather than silently truncating the blueprint.
- Persist an immutable audit record for every external model HTTP attempt so operators can identify which user/job/batch succeeded or failed, when it ran, how long it took, which provider/model handled it, token usage, and the bounded failure reason.
- Aggregate administrator Token metrics from that invocation ledger, and refresh the aggregate whenever recent invocation history is refreshed, so the displayed totals follow the requests that actually reached the provider.
- Add job status, progress phase, preview ID, preview version, expiry, source, phase timings, and bounded failure metadata.
- Prevent a late AI result from silently overwriting a preview the user has already edited; expose the newer version for explicit review instead.
- Separate the short interactive response budget from a configurable background model budget. Default the model job budget to 5 minutes and allow administrators to select a 5-minute or 10-minute tier.
- Size the model output budget from the requested plan scope and reuse safe provider transports without weakening public-origin enforcement.
- Preserve the existing `/api/ai/*` routes with additive job metadata while introducing a status endpoint for polling.

## Capabilities

### Modified Capabilities

- `ai-plan-generator`: Make the model a real task-decomposition collaborator through asynchronous planning jobs while the Agent remains the final authority.
- `plan-management`: Schedule, version, review, and transactionally commit local or AI-decomposed previews with the same authoritative rules.
- `admin-config`: Configure and observe separate interactive and background model budgets, provider health, completion rate, latency, and fallback reasons.

## Confirmed Decisions

- AI owns semantic learning decomposition: stages, task intent, task count suggestions, objectives, descriptions, difficulty, estimated effort, ordering, and prerequisite hints.
- The backend Agent owns account context, privacy boundaries, dates, time slots, workload limits, conflict repair, derived totals, final validation, and persistence.
- The initial API response returns the local baseline promptly instead of holding a mobile request open for the full model call.
- Background model decomposition defaults to a 5-minute deadline and is configurable as a 5-minute or 10-minute tier.
- The model returns a compact blueprint without persisted IDs or authoritative dates and times.
- The backend may split an oversized blueprint task into sequenced parts when it cannot fit within the user's daily capacity, and reports that repair in warnings.
- AI results create a new preview version. They auto-replace the displayed baseline only when the user has not edited or committed an older version.
- Users may accept the local baseline without waiting for AI, continue waiting, or explicitly review an AI version that arrives later.
- Local planning remains available when model decomposition is disabled or unavailable, but an AI job remains truthful (`retrying`/failed) until a valid AI preview is published or explicitly cancelled.

## Non-Goals

- The model does not own final scheduling, conflict resolution, workload enforcement, or database writes.
- This change does not add an open-ended chatbot or streaming token UI.
- This change does not require a distributed queue; a persistent database-backed job runner is sufficient for the current deployment scale.
- This change does not redesign the provider's general retry policy beyond what is necessary for safe connection reuse.
- Broad fault-injection matrices and unrelated planning-page enhancements are deferred once the functional completion paths in this change are verified.
