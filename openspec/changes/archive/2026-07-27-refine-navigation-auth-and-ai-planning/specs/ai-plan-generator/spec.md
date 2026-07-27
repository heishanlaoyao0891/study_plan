# ai-plan-generator Delta Specification

## MODIFIED Requirements

### Requirement: AI generates a study plan from user description
The system SHALL accept a natural-language learning goal and planning constraints as a durable asynchronous generation job, SHALL validate model-assisted output with the backend planning Agent, and SHALL directly persist the resulting plan and tasks without requiring a client preview commit.

#### Scenario: Submit plan generation
- **WHEN** an authenticated user submits a valid goal, availability, start date, skip dates, and optional additional instructions
- **THEN** the system persists a generation job and promptly returns its identifier and current status without waiting for model completion

#### Scenario: Generate an enriched plan
- **WHEN** a worker produces a valid local candidate and model enrichment succeeds within its execution budget
- **THEN** the system revalidates and persists the plan with source `local_enriched`

#### Scenario: Generate without model enrichment
- **WHEN** enrichment is disabled, unavailable, quota-limited, invalid, or times out but local planning succeeds
- **THEN** the system persists the valid local plan with source `local` and records truthful generation metadata

#### Scenario: Generation completes
- **WHEN** the final candidate passes current schedule, workload, ownership, and structural validation
- **THEN** plan creation, task creation, and the job's succeeded state with result plan ID are committed atomically

### Requirement: User can edit AI-generated plan
The system SHALL allow users to edit the normally persisted plan and its tasks after asynchronous generation succeeds rather than editing an AI preview before commit.

#### Scenario: Open generated plan
- **WHEN** a generation job succeeds and the user opens its resulting plan
- **THEN** the normal plan detail and edit workflows allow permitted plan and task changes

#### Scenario: User leaves during generation
- **WHEN** the user navigates away or closes the client after submitting a generation job
- **THEN** generation continues independently and the completed plan remains available through the normal plan list

### Requirement: AI generation has fallback behavior
The system SHALL execute local planning as a first-class stage inside the asynchronous job and SHALL expose terminal failure only when no valid plan can be safely persisted.

#### Scenario: Provider is unavailable
- **WHEN** the external provider times out or fails after a valid local candidate exists
- **THEN** the worker persists the local candidate and marks the job succeeded with truthful fallback metadata

#### Scenario: No valid candidate can be persisted
- **WHEN** input, planning, final conflict checks, or persistence cannot produce a valid plan after bounded retries
- **THEN** the system marks the job failed with a safe actionable message and creates no partial plan or tasks

## ADDED Requirements

### Requirement: AI generation jobs are durable and observable
The system SHALL persist generation state and SHALL allow an authenticated user to observe only their own current or identified jobs across page navigation, client restart, and backend process restart.

#### Scenario: Client observes active generation
- **WHEN** a user's job is pending or running
- **THEN** the plan and AI entry interfaces show that AI is generating and prevent accidental duplicate submission

#### Scenario: Client resumes observation
- **WHEN** the client reopens while its generation job is active
- **THEN** it restores status from the server and continues observing without restarting generation

#### Scenario: Worker restarts
- **WHEN** a backend process stops while a job is pending or holds an expired running lease
- **THEN** a worker can safely claim or reclaim the durable job and continue within the bounded attempt policy

#### Scenario: User requests another active job
- **WHEN** a user submits generation while they already have a pending or running job
- **THEN** the system returns the existing active job instead of creating concurrent duplicate work

#### Scenario: User requests another user's job
- **WHEN** an authenticated user requests a generation job owned by another user
- **THEN** the system returns not found or forbidden without exposing its inputs, status, or result

### Requirement: Additional instructions guide generation within constraints
The system SHALL accept optional free-form `追加说明` as user planning preferences and SHALL keep backend-owned safety, schedule, workload, and validation rules authoritative.

#### Scenario: User provides detailed preferences
- **WHEN** a user submits additional instructions such as reduced weekend work or an emphasis on practical exercises
- **THEN** the planning Agent includes those preferences in bounded content generation where they do not conflict with authoritative constraints

#### Scenario: Additional instructions contradict constraints
- **WHEN** additional instructions request unavailable, conflicting, unauthorized, or otherwise invalid behavior
- **THEN** the system ignores or repairs the conflicting preference and persists only a validated plan

## REMOVED Requirements

### Requirement: Editable preview provenance is bound
**Reason**: The asynchronous worker persists validated output directly, so generated candidates no longer cross a client trust boundary before commit.

**Migration**: Clients submit a generation job, observe its status, and edit the resulting persisted plan through normal plan APIs.

### Requirement: Preview commit is concurrency-safe
**Reason**: There is no client preview commit in the new flow; concurrency control moves to job idempotency, lease claiming, and atomic result-plan persistence.

**Migration**: Replace preview commit idempotency keys with generation-job submission idempotency and result plan binding.
