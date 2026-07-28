# Tasks

## 1. Agent Contract and Interactive Response

- [x] Define typed planning request, preview, source, warnings, enrichment status, and phase timings.
- [x] Validate goal length, plan duration, dates, skip dates, availability, desired hours, and refinements.
- [x] Separate local planning availability from external model quota.
- [x] Add persistent preview ID, version, expiry, context fingerprint, and planning job metadata to generation responses.
- [x] Separate the 1-5 second interactive baseline target from the 5-minute or 10-minute background model budget.
- [x] Add authenticated job-status polling with typed queued, decomposing, scheduling, ready, fallback, cancelled, and expired states.

## 2. Deterministic Planning and Scheduling Core

- [x] Extract reusable occupancy, schedule validation, overload, and candidate-allocation rules from handlers.
- [x] Load exact occupied intervals and safe aggregate learning context.
- [x] Maintain learning, reading, exam, project, and general local stage templates.
- [x] Build progressive local tasks with review cadence and profile-based pacing.
- [x] Reconcile desired hours with availability and emit capacity warnings.
- [x] Search alternate time slots and eligible dates to repair avoidable conflicts.
- [x] Produce stable ordered local tasks and recompute derived totals.
- [x] Schedule a variable number of AI blueprint tasks across eligible dates, including multiple short tasks per day.
- [x] Split oversized blueprint tasks into ordered parts while preserving intent and prerequisite order.

## 3. Persistent Planning Jobs

- [x] Add planning job and immutable preview-version models, indexes, and migrations.
- [x] Create model jobs transactionally with the local baseline and prevent duplicate active jobs for equivalent requests.
- [x] Implement bounded worker claiming, leases, attempts, cancellation, and terminal state transitions.
- [x] Recover abandoned running jobs after restart without publishing duplicate preview versions.
- [x] Expire jobs and preview versions after a bounded retention period.
- [x] Enforce user ownership and privacy on job and preview lookups.

## 4. AI Task Decomposition

- [x] Define a bounded structured blueprint schema for plan metadata, stages, variable task count, effort, difficulty, order, and prerequisite hints.
- [x] Replace the complete scheduled-plan prompt with a compact normalized decomposition brief.
- [x] Calculate a bounded output token allowance from plan scope instead of using a fixed 1024-token limit.
- [x] Run decomposition in the background worker with a default 5-minute deadline configurable as a 5-minute or 10-minute tier.
- [x] Reuse safe provider transports and idle connections per validated provider origin.
- [x] Remove redundant per-attempt DNS validation while retaining configuration-time and dial-time public-address enforcement.
- [x] Validate blueprint schema, task bounds, prerequisite references, and privacy before scheduling.
- [x] Schedule and revalidate a valid blueprint before publishing source `ai_decomposed`.
- [x] Meter only actual external provider attempts and keep local planning outside model quota.
- [x] Reuse strict encrypted provider configuration and bounded public-origin validation.
- [x] Record provider/model, attempts, p50/p95 latency inputs, token usage where available, and bounded fallback reasons.

## 5. Preview Version and Commit Integrity

- [x] Persist immutable local and AI-decomposed preview versions and bind jobs to their baseline and result versions.
- [x] Bind immutable plan metadata and ordered opaque task identities in signed user-scoped provenance.
- [x] Add bounded SQLite race retries, same-payload replay, and different-payload conflict behavior.
- [x] Reapply overload and schedule validation transactionally at commit.
- [x] Recompute sorted dates, estimated minutes, total hours, and weekly target server-side.
- [x] Extend provenance and commit checks to preview ID, version, expiry, and context fingerprint.
- [x] Add server-side derived-version mutations for task add, remove, split, and reorder operations.
- [x] Publish an AI version without mutating a dirty, committed, stale, or expired local version.
- [x] Persist and display `manual`, `local`, and `ai_decomposed` source values.

## 6. User and Admin Experience

- [x] Rename the user entry to smart plan generation and explain optional AI participation.
- [x] Mutually exclude generation and commit, clear stale previews at generation start, and ignore stale synchronous responses.
- [x] Show local source, warnings, provider outcome, and safe fallback reason.
- [x] Show immediate local baseline plus queued, decomposing, scheduling, ready, and fallback progress states.
- [x] Poll job status with cancellation on page exit and bounded retry/backoff on transient client errors.
- [x] Auto-display an AI version only when the local baseline is untouched.
- [x] Preserve local edits and offer explicit local-versus-AI version review when a late result arrives.
- [x] Allow the user to commit the local baseline while AI runs and ignore later mutation of that committed plan.
- [x] Add start date, skip dates, refinement, task add, task remove, and reorder controls compatible with preview versions.
- [x] Rename admin enablement to model decomposition and expose interactive target plus background job budget.
- [x] Add admin queue depth, success/fallback rate, p50/p95 latency, token usage, and bounded failure metrics.

## 7. Verification

- [x] Add local planning tests for each stage template and profile pacing.
- [x] Add hours/availability, skip-date, occupancy, alternate-slot, and conflict-repair tests.
- [x] Add disabled, quota, timeout, cancellation, invalid-output, and provider-failure characterization tests proving local success.
- [x] Add semantic merge characterization tests proving model output cannot override final constraints in the current path.
- [x] Add existing provenance substitution, concurrent idempotency, overload, and recomputation tests.
- [x] Add blueprint tests proving AI can vary stages and task count while invalid schemas are rejected.
- [x] Add scheduling tests for variable task counts, multiple tasks per day, oversized splits, prerequisites, occupancy, and workload repair.
- [x] Add a deadline regression test proving the configured multi-minute background budget is not truncated by the interactive request context.
- [x] Add derived preview tests for add, remove, split, reorder, committed/expired rejection, and concurrent version creation.
- [x] Add provider transport reuse, redirect, DNS rebinding, and configuration-refresh tests.
- [x] Run backend tests, frontend type checks, H5, mini-program, and admin builds after implementation.
- [x] Strictly validate the revised OpenSpec change before implementation handoff and again before archive.

## 8. Durable AI Generation Completion

- [x] Add durable outline, batch-expansion, validation, repair, scheduling, publishing, and retry-wait phases with persisted checkpoints and restart resume.
- [x] Normalize safe blueprint deviations including opaque IDs, stage-local task ordering, supported difficulty aliases, unknown fields, and effort bounds.
- [x] Detect provider truncation and malformed JSON and retry failed batches with precise diagnostics, adjusted token allowance, or smaller batches.
- [x] Constrain total blueprint effort to requested capacity and support complete plans through 30 days.
- [x] Prevent invalid provider output or local fallback from being published or reported as `ai_decomposed` success.
- [x] Separate provider-attempt accounting from user quota and charge exactly once after successful AI preview publication.
- [x] Add a versioned Prompt Playbook/error-pattern registry and include active preventive guidance in future prompts and admin metrics.
- [x] Update frontend job states so recoverable failures continue polling through retry wait and page exit does not cancel server work.
- [x] Add regression coverage for truncation, targeted repair, normalization, capacity, 30-day batching, checkpoint resume, truthful provenance, and successful-publication quota.
- [x] Run backend, frontend, admin, strict OpenSpec validation, update the living project scan, and push the completed change.

## 9. AI Invocation Audit Ledger

- [x] Add the immutable invocation-log model, migration, indexes, retention-ready fields, and safe bounded error taxonomy.
- [x] Propagate user/job/phase/batch/repair context through Agent calls and record every provider HTTP retry from start through completion.
- [x] Store only fingerprints, sizes, timings, HTTP/finish metadata, and token usage; never raw prompts, responses, credentials, or private learning records.
- [x] Add paginated/filterable admin APIs and an invocation-history table showing who succeeded, who failed, and why.
- [x] Add regression tests for success, HTTP failure, timeout/truncation, internal retry, context linkage, redaction, and audit-write failure behavior.
- [x] Run backend/frontend/admin validation, strict OpenSpec validation, update the living scan, commit, and push.
