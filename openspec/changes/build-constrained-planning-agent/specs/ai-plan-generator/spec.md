# ai-plan-generator Delta Specification

## MODIFIED Requirements

### Requirement: AI generates a study plan from user description
The system SHALL use a backend planning Agent to return a valid local baseline promptly and SHALL allow the configured model to asynchronously decompose the goal into an AI-enhanced plan version.

#### Scenario: Return an immediate baseline
- **WHEN** the user submits a valid planning request
- **THEN** the Agent returns a conflict-free local preview with preview ID, version, source `local`, and decomposition status without waiting for the full model deadline

#### Scenario: Generate an AI-decomposed version
- **WHEN** the background model job returns a valid learning blueprint within its configured budget
- **THEN** the Agent schedules and revalidates the blueprint and publishes a new preview version with source `ai_decomposed`

#### Scenario: User accepts the baseline early
- **WHEN** the user commits the valid local preview while model decomposition is still queued or running
- **THEN** the system creates the local plan and prevents a later model result from replacing the committed plan

#### Scenario: Generate without model decomposition
- **WHEN** decomposition is disabled, unavailable, quota-limited, invalid, or times out
- **THEN** the local preview remains usable and the job exposes a truthful terminal fallback status

### Requirement: AI asks for user availability before generating
The system SHALL validate desired plan duration, daily capacity, available time ranges, start date, skip dates, and current schedule occupancy before creating the local baseline or model job.

#### Scenario: Desired hours exceed availability
- **WHEN** requested daily hours exceed the supplied daily time range
- **THEN** the Agent uses actual available capacity and returns an explanatory warning

#### Scenario: Requested time is occupied
- **WHEN** requested availability intersects existing unfinished work
- **THEN** the Agent searches another valid time or eligible date and returns a repaired local schedule rather than an avoidable conflict

### Requirement: AI considers historical learning ability
The system SHALL provide only safe aggregate learning history to local pacing and model decomposition and SHALL not expose raw private learning records in the provider prompt.

#### Scenario: User has lower recent completion
- **WHEN** recent completion or postponement signals indicate overload risk
- **THEN** the local baseline reduces density and the model brief requests smaller tasks, review cadence, and buffer capacity

### Requirement: AI usage is controlled
The system SHALL meter external model job attempts independently of local planning and SHALL bound concurrent and daily model usage per user.

#### Scenario: Model quota is exhausted
- **WHEN** a user has no remaining decomposition quota
- **THEN** local planning still succeeds and no model job is started

#### Scenario: User submits duplicate generation requests
- **WHEN** equivalent requests are submitted while a matching non-terminal job exists
- **THEN** the system reuses or rejects the duplicate job without consuming unbounded additional provider attempts

### Requirement: AI generation has fallback behavior
The system SHALL keep the local baseline available throughout the asynchronous model lifecycle and SHALL never require model success to create a valid plan.

#### Scenario: Model exceeds the background deadline
- **WHEN** decomposition does not complete within the configured 5-minute or 10-minute model budget
- **THEN** the provider request is cancelled, the job becomes `fallback`, and the local baseline remains available

#### Scenario: Backend restarts during decomposition
- **WHEN** a claimed job loses its worker lease during a process restart
- **THEN** the system returns the job to the queue at most once or completes it as fallback without creating duplicate preview versions

## ADDED Requirements

### Requirement: AI decomposes the learning goal into a task blueprint
The model SHALL produce a bounded structured blueprint containing ordered learning stages and a variable number of concrete tasks rather than merely rewriting a fixed local task list.

#### Scenario: Model chooses meaningful stages
- **WHEN** a user requests a multi-day programming, reading, exam, project, or general learning plan
- **THEN** the blueprint contains goal-specific stage names, ordered task objectives, estimated effort, difficulty, and prerequisite hints

#### Scenario: Model varies task count
- **WHEN** the learning goal requires more or fewer semantic tasks than the requested plan duration
- **THEN** the blueprint may return a different task count while remaining within configured schema and size limits

#### Scenario: Model attempts to provide authoritative schedule fields
- **WHEN** model output includes persisted IDs or untrusted final dates and time ranges
- **THEN** the Agent ignores or rejects those fields and schedules the semantic blueprint itself

### Requirement: Planning Agent owns final constraints
The planning Agent SHALL own dates, time slots, workload limits, occupancy, conflict repair, derived totals, final validation, and persistence for local and AI-decomposed previews.

#### Scenario: Blueprint contains an oversized task
- **WHEN** a task cannot fit within the user's daily capacity
- **THEN** the Agent splits it into ordered parts or falls back with a bounded warning rather than persisting an impossible schedule

#### Scenario: Blueprint tasks exceed available dates
- **WHEN** the ordered tasks cannot fit in the requested range without conflict
- **THEN** the Agent uses later eligible dates when allowed or rejects the AI version while preserving the local baseline

#### Scenario: Blueprint contains prerequisites
- **WHEN** task B depends on task A
- **THEN** the final schedule places A before B after all packing and conflict repair

### Requirement: Local planner uses progressive stage templates
The system SHALL retain maintained learning, reading, exam, project, and general templates as a fast independent baseline and as safe hints for model decomposition.

#### Scenario: Model is unavailable
- **WHEN** no model job can be started or completed
- **THEN** the user still receives a progressive local plan suitable for editing and commit

### Requirement: Planning jobs are observable
The system SHALL expose user-scoped job status, current phase, provider/model, bounded failure reason, attempts, timings, expiry, and newest available preview version.

#### Scenario: Poll a running job
- **WHEN** the client requests the status of its queued, decomposing, or scheduling job
- **THEN** the response reports the current phase without exposing raw provider errors, credentials, or another user's data

#### Scenario: AI version becomes ready
- **WHEN** scheduling and validation of the blueprint succeed
- **THEN** the job becomes `ready` and references an immutable AI-decomposed preview version

#### Scenario: Poll another user's job
- **WHEN** a user requests a job they do not own
- **THEN** the system returns not found or forbidden without disclosing job metadata

### Requirement: Model output budget scales with plan scope
The system SHALL calculate a bounded model output allowance from requested duration and expected blueprint size instead of using one fixed token limit for every plan.

#### Scenario: Generate a longer plan
- **WHEN** the user requests a plan with many learning days
- **THEN** the model receives a larger bounded output allowance than a one-day plan while remaining below the configured provider safety limit

### Requirement: Editable preview versions are bound
The system SHALL store immutable user-scoped preview versions with expiry, context fingerprint, task identities, source, and signed provenance.

#### Scenario: AI result arrives before user edits
- **WHEN** the AI version becomes ready while the displayed local baseline is untouched
- **THEN** the client may replace the displayed baseline with the newer version

#### Scenario: AI result arrives after user edits
- **WHEN** the user has edited the local baseline before the AI version becomes ready
- **THEN** the client preserves the edits and offers the AI version for explicit review instead of overwriting them

#### Scenario: User changes preview structure
- **WHEN** the user adds, removes, splits, or reorders preview tasks
- **THEN** the server creates a derived preview version with fresh ordered task identities and provenance before commit

#### Scenario: Commit a stale or expired version
- **WHEN** the client submits a preview version that is stale, expired, or does not match its provenance
- **THEN** commit returns a typed conflict before persistence

### Requirement: Preview commit is concurrency-safe
The system SHALL reconcile bounded SQLite busy and uniqueness races by user, preview version, idempotency key, and normalized committed payload.

#### Scenario: Identical commits race
- **WHEN** concurrent requests use the same user, preview version, idempotency key, and payload
- **THEN** exactly one plan is created and successful replays return that plan

#### Scenario: AI finishes during commit
- **WHEN** an AI version is published while the user is committing an older valid local version
- **THEN** the local commit completes against its immutable version and the AI result cannot mutate the created plan
