# Tasks

## 1. Agent Contract and Observability
- [x] Define typed planning request, preview, source, enrichment status, warnings, and phase timings.
- [x] Validate goal length, day count, dates, skip dates, availability, desired hours, and refinements.
- [x] Separate local planning usage from external enrichment quota and statuses.
- [x] Apply one advertised request budget to context queries, quota checks, validation, and enrichment while reserving local response time.

## 2. Deterministic Planning Core
- [ ] Extract reusable occupancy, schedule validation, and overload rules from handlers.
- [x] Load exact occupied intervals and safe aggregate learning context.
- [x] Add learning, reading, exam, project, and general stage templates.
- [x] Build progressive tasks with review cadence and real profile-based pacing.
- [x] Reconcile desired hours with availability and emit capacity warnings.
- [x] Search alternate time slots and eligible dates to repair avoidable conflicts.
- [x] Produce stable ordered tasks and recompute all derived totals locally.

## 3. Bounded Model Collaboration
- [x] Send SiliconFlow the normalized brief and canonical local skeleton.
- [x] Restrict enrichment to allowed semantic fields and bounded schedule suggestions.
- [x] Merge enrichment onto the local candidate and repair or reject invalid suggestions.
- [x] Return the local candidate on disabled, quota, timeout, provider, or output failures.
- [x] Include refinement instructions in actual regeneration requests.

## 4. Preview and Commit Integrity
- [ ] Add user-scoped expiring preview ID, version, context snapshot, and provenance.
- [x] Bind immutable plan metadata and ordered opaque task identities in signed provenance without trusting a client original preview.
- [x] Add bounded SQLite race retries, same-payload replay, and different-payload conflict behavior.
- [x] Reapply overload and schedule validation transactionally at commit.
- [x] Recompute sorted dates, estimated minutes, total hours, and weekly target server-side.
- [x] Persist manual/local/local_enriched source without mislabeling fallback plans.

## 5. User and Admin Experience
- [x] Rename the user entry to smart plan generation and explain optional AI enhancement.
- [x] Mutually exclude generation and commit, clear stale previews at generation start, and ignore stale generation responses.
- [x] Show local/enriched source, warnings, enrichment outcome, and safe fallback reason.
- [ ] Add start date, skip dates, regenerate refinement, task add, and reorder controls.
- [ ] Rename admin enablement to model enrichment and expose the interactive budget.

## 6. Verification
- [x] Add local planning tests for each stage template and profile pacing.
- [x] Add hours/availability, skip-date, occupancy, alternate-slot, and conflict-repair tests.
- [x] Add disabled, quota, timeout, cancellation, invalid-output, and provider-failure tests proving local success.
- [x] Add enrichment merge tests proving model output cannot override final constraints.
- [x] Add provenance substitution, concurrent idempotency, overload, and recomputation tests.
- [ ] Add frontend checks for loading, source, warnings, regeneration, edits, and duplicate taps.
- [x] Run backend tests, frontend type check, H5 and mini-program builds, and strict OpenSpec validation.
- [x] Strictly validate this OpenSpec change.
