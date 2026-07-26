## Context

The backend already validates planning input, loads safe aggregate learning history, finds exact occupied intervals, builds conflict-free local tasks, validates schedule unions, returns fallback metadata, and commits previews transactionally. The current model collaboration is still synchronous: the request advertises a 12-second total budget and cancels enrichment after eight seconds. Provider configuration may allow 30-120 seconds, but the shorter child context always wins.

The current prompt sends the complete local plan and requires the model to preserve task count, order, dates, and time ranges. Merge logic rejects any changed task count or order and accepts only title, objective, description, and difficulty. This makes the model a semantic rewriter rather than a task-decomposition collaborator. A large model may also need more than eight seconds to return a complete multi-day JSON object, especially after DNS, TCP, TLS, and provider queue latency.

## Decisions

### Planning has an immediate baseline and an asynchronous AI version

`POST /api/ai/generate-plan` continues to validate the request, load context, create a conflict-free local baseline, assign a preview ID and version, and return promptly. When model decomposition is enabled and quota is available, the same request also creates a persistent planning job and returns its ID and initial state.

The client displays the baseline immediately with `source=local` and a visible `decomposition_status`. It polls a job-status resource while the user remains on the preview page. The user may commit the local baseline without waiting.

The interactive response target is distinct from the model deadline. Its default is two seconds and its configurable range is one to five seconds. Missing that target is an operational warning; failure to build a valid local baseline remains a request failure.

### AI produces a learning blueprint, not a persisted schedule

The provider receives:

- the normalized goal and refinement;
- target plan duration, daily capacity, skip dates, and high-level slot inventory;
- safe aggregate learning profile and workload signals;
- the locally classified goal type and optional stage hints;
- a compact output contract.

The model returns a blueprint containing plan title, summary, rationale, ordered stages, and ordered tasks. Each task contains stage, title, concrete objective, description, estimated minutes, difficulty, and optional prerequisite references. The model may vary the task count and stage boundaries. It does not return database IDs or authoritative dates and time ranges.

The output token allowance is calculated from requested plan scope and bounded by provider-safe minimum and maximum values. The contract remains structured JSON and requests non-thinking mode when supported.

### The backend turns the blueprint into an executable plan

After validating the blueprint schema and content bounds, the Agent allocates tasks into the user's available dates and time ranges. It uses the same occupancy, overlap, overload, and mutation rules as manual plans and recovery scheduling.

The scheduler may:

- place multiple short tasks on one eligible date when capacity permits;
- move work to a later eligible date;
- split an oversized task into ordered parts when it cannot fit in a daily slot;
- preserve prerequisite ordering;
- reject or repair invalid estimated effort or difficulty values;
- return warnings describing material repairs.

The model never bypasses final schedule validation. If the blueprint cannot be repaired into a valid plan, the job falls back to the local baseline with a bounded reason.

### Planning jobs are persistent and observable

Each job is user-scoped and stores request identity, status, current phase, provider/model, baseline preview ID and version, result version, timestamps, expiry, attempt count, bounded error reason, and phase timings. Job states are:

`queued -> decomposing -> scheduling -> ready`

Terminal alternatives are `fallback`, `cancelled`, and `expired`.

An in-process worker claims queued jobs transactionally. A process restart returns abandoned running jobs to `queued` once when their lease expires. Jobs and preview versions expire after a bounded retention window.

The job-status response never exposes raw provider errors, credentials, prompts containing private records, or other users' data.

### Preview versions prevent late-result overwrite

The local baseline is preview version 1. A successful AI decomposition produces a new immutable version with `source=ai_decomposed`. Every version carries task identities, request/context fingerprint, creation time, and expiry.

The client tracks the version it displayed and whether the user has edited it. If the AI version arrives while the baseline is untouched, the client may replace it automatically. If the baseline was edited, the AI version is presented as a separate review option. A late result never silently overwrites user edits or a committed plan.

Stored preview versions remain immutable. Field edits may be submitted as a draft derived from one base version and are revalidated at commit. Adding, removing, splitting, or reordering tasks uses a server-side preview mutation that creates a new derived version with fresh ordered task identities and provenance. The client cannot invent identities or change version structure only in the final commit payload.

Commit requires preview ID, version, provenance, and idempotency key. The backend revalidates current occupancy and workload transactionally. A stale or expired version returns a typed conflict rather than being trusted.

### Model timeout is a background operational control

The model job deadline defaults to 60 seconds and is configurable from 15 to 120 seconds. This budget includes provider validation, connection establishment, request execution, response reading, and bounded transient retry.

Interactive response budget and background model budget are stored and reported separately. Changing the background budget affects new jobs and does not extend an already running job.

### Provider transport is safe and reusable

The provider layer reuses a transport per validated provider origin so normal requests can reuse DNS results and idle TCP/TLS connections. Configuration save/test still validates that the destination resolves only to public addresses. Dial-time public-address validation remains in place to protect against DNS rebinding, but redundant validation inside one generation attempt is removed or cached for the attempt.

Redirects remain restricted to the validated origin. Idle connections and cached provider clients are discarded when provider URL, credential version, or relevant transport configuration changes.

### Compatibility is additive

Existing generation and commit routes remain available. Generation responses add job and preview-version metadata. A job-status route returns state and the newest available preview version. Existing clients that ignore job metadata continue to receive and commit the local baseline.

A future route family may use neutral planning resources, but route renaming is not required for this change.

## Risks / Trade-offs

- AI results can arrive after the user starts editing -> preserve immutable versions and require explicit review when the baseline is dirty.
- Persistent jobs add lifecycle and restart complexity -> use database leases, bounded retries, expiry, and idempotent state transitions.
- Variable task counts make scheduling more complex -> keep all final constraints in the Agent and add property/boundary tests for packing, splitting, ordering, and conflicts.
- Longer model budgets increase cost and queue occupancy -> retain per-user quota, cap concurrency, record latency/token usage, and allow users to commit the local baseline immediately.
- Model decomposition can still be generic -> include goal-specific refinement, safe learning profile signals, and editable results while retaining local fallback.
- Transport reuse must not weaken SSRF protection -> cache only validated origins and retain dial-time public-IP enforcement.

## Migration Plan

1. Add persistent preview IDs, versions, context fingerprints, and planning job records without changing existing local generation behavior.
2. Add job creation and polling metadata to existing generation responses and update the frontend to display baseline/job states.
3. Define and validate the compact AI blueprint contract and dynamic output budget.
4. Add the worker, provider transport reuse, background timeout, retry policy, and job observability.
5. Add blueprint-to-schedule allocation, packing, splitting, repair warnings, and final validation.
6. Produce immutable AI preview versions and handle untouched, edited, committed, stale, and expired baseline states.
7. Update admin controls and metrics for interactive target, background timeout, success rate, p50/p95 latency, token usage, and fallback reasons.
8. Remove the synchronous semantic-only enrichment path after compatible clients have migrated.

## Deferred Work

- A broader redesign of provider retry classification and jitter policy is outside this change once the background deadline and safe reusable transport are working.
- Exhaustive fault-injection suites that do not directly verify authoritative scheduling, preview mutations, transport safety, metrics, or the user workflow are deferred to a later quality change.
