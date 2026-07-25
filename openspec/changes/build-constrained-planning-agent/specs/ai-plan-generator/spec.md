# ai-plan-generator Delta Specification

## MODIFIED Requirements

### Requirement: AI generates a study plan from user description
The system SHALL use a backend planning Agent to create a valid local plan candidate and MAY use the configured model to enrich that candidate within bounded constraints.

#### Scenario: Generate an enriched plan
- **WHEN** the Agent has produced a valid candidate and model enrichment succeeds within budget
- **THEN** the system returns a revalidated editable preview with source `local_enriched`

#### Scenario: Generate without model enrichment
- **WHEN** enrichment is disabled, unavailable, quota-limited, invalid, or times out
- **THEN** the system returns the valid local preview with source `local` and a truthful enrichment status

### Requirement: AI asks for user availability before generating
The system SHALL validate and reconcile desired daily hours, available time ranges, start date, skip dates, and current schedule occupancy before model collaboration.

#### Scenario: Desired hours exceed availability
- **WHEN** requested daily hours exceed the supplied daily time range
- **THEN** the Agent uses available capacity, adjusts duration or plan length where possible, and returns an explanatory warning

#### Scenario: Requested time is occupied
- **WHEN** the requested time intersects existing unfinished work
- **THEN** the Agent searches another valid time or eligible date and returns the repaired schedule rather than an avoidable conflict

### Requirement: AI considers historical learning ability
The system SHALL use safe aggregate learning history to change actual local task pacing, review cadence, and buffer placement before enrichment.

#### Scenario: User has lower recent completion
- **WHEN** recent completion or postponement signals indicate overload risk
- **THEN** the Agent reduces local task density or duration and adds review or buffer capacity

### Requirement: AI usage is controlled
The system SHALL meter only external model enrichment attempts and SHALL keep local planning available independently of model quota.

#### Scenario: Model quota is exhausted
- **WHEN** a user has no remaining enrichment quota
- **THEN** local planning still succeeds and no additional model request is made

### Requirement: AI generation has fallback behavior
The system SHALL treat local planning as a first-class capability rather than constructing it only after model failure.

#### Scenario: Model exceeds interactive deadline
- **WHEN** enrichment does not complete within the Agent's total model budget
- **THEN** the provider request is cancelled and the valid local candidate is returned before the API request budget expires

## ADDED Requirements

### Requirement: Planning Agent owns final constraints
The planning Agent SHALL own dates, time slots, workload limits, conflict repair, derived totals, final validation, and persistence.

#### Scenario: Model suggests invalid schedule changes
- **WHEN** model output moves work outside availability or into a conflicting slot
- **THEN** the Agent rejects or repairs those changes before returning the preview

### Requirement: Local planner uses progressive stage templates
The system SHALL decompose goals with maintained learning, reading, exam, project, and general stage templates.

#### Scenario: Generate a project plan locally
- **WHEN** a goal is classified as project delivery
- **THEN** the local candidate progresses through requirements, setup, milestones, integration, and review

### Requirement: Planning response is observable
The system SHALL return source, enrichment status, warnings, bounded failure reason, phase timing metadata, and the advertised overall request budget.

#### Scenario: Local result follows provider timeout
- **WHEN** a local preview is returned after enrichment timeout
- **THEN** the client can explain that the plan is valid locally generated work and that AI enhancement did not complete

### Requirement: Editable preview provenance is bound
The system SHALL bind each generated candidate's immutable plan metadata, task count, and ordered opaque task identities in a signed, user-scoped, expiring token without trusting a client-supplied original preview.

#### Scenario: User edits an allowed preview field
- **WHEN** the user edits a task date, start/end time, title, objective, description, or difficulty while preserving task identities and count
- **THEN** commit revalidates the schedule, recomputes derived values, and may persist the edited candidate

#### Scenario: Client substitutes unrelated tasks
- **WHEN** commit removes, adds, reorders, or replaces a signed task identity, or changes signed plan metadata
- **THEN** commit rejects the candidate before persistence

### Requirement: Preview commit is concurrency-safe
The system SHALL reconcile bounded SQLite busy and uniqueness races by user, idempotency key, and normalized committed payload.

#### Scenario: Identical commits race
- **WHEN** concurrent requests use the same user, idempotency key, and payload
- **THEN** exactly one plan is created and every successful replay returns that plan

#### Scenario: An idempotency key is reused for another payload
- **WHEN** the same user and key are submitted with a different normalized preview
- **THEN** the system returns conflict rather than an internal error
