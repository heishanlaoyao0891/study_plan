# ai-plan-generator Specification

## Purpose
TBD - created by archiving change study-checkin-miniapp. Update Purpose after archive.
## Requirements
### Requirement: AI generates a study plan from user description
The system SHALL accept a natural-language learning goal and planning constraints as a durable asynchronous generation job, SHALL validate model-assisted output with the backend planning Agent, and SHALL directly persist the resulting plan and tasks without requiring a client preview commit.

#### Scenario: Submit plan generation
- **WHEN** an authenticated user submits a valid goal, availability, start date, skip dates, and optional additional instructions
- **THEN** the system persists a generation job and promptly returns its identifier and current status without waiting for model completion

#### Scenario: Generate a model-decomposed plan
- **WHEN** a worker produces a valid local fallback candidate and model task decomposition succeeds within its execution budget
- **THEN** the backend schedules and revalidates the model-defined stages, task count, objectives, effort, and order, then persists the plan with source `ai_decomposed`

#### Scenario: Generate without model decomposition
- **WHEN** model decomposition is disabled, unavailable, quota-limited, invalid, or times out but local planning succeeds
- **THEN** the system persists the valid local plan with source `local` and records truthful generation metadata

#### Scenario: Generation completes
- **WHEN** the final candidate passes current schedule, workload, ownership, and structural validation
- **THEN** plan creation, task creation, and the job's succeeded state with result plan ID are committed atomically

### Requirement: AI asks for user availability before generating
The system SHALL collect user's daily available hours and time slots before AI generation.

#### Scenario: Provide availability
- **WHEN** user starts AI plan generation
- **THEN** system prompts user for: daily available hours, preferred time slot (e.g. 20:00-22:00), start date, and dates to skip

### Requirement: AI estimates total duration based on goal
The system SHALL estimate the total study duration for the goal and distribute it across days.

#### Scenario: Estimate and distribute
- **WHEN** AI receives a goal like "learn Go"
- **THEN** AI estimates total hours needed (e.g. 60h) and generates daily tasks spread across days based on daily available hours

### Requirement: AI considers historical learning ability
The system SHALL optionally use user's historical study data to adjust plan difficulty and pace.

#### Scenario: Personalized pace
- **WHEN** AI generates plan and user has historical data
- **THEN** AI adjusts daily task density based on user's past completion rate

### Requirement: User can edit AI-generated plan
The system SHALL allow users to edit the normally persisted plan and its tasks after asynchronous generation succeeds rather than editing an AI preview before commit.

#### Scenario: Open generated plan
- **WHEN** a generation job succeeds and the user opens its resulting plan
- **THEN** the normal plan detail and edit workflows allow permitted plan and task changes

#### Scenario: User leaves during generation
- **WHEN** the user navigates away or closes the client after submitting a generation job
- **THEN** generation continues independently and the completed plan remains available through the normal plan list

### Requirement: Plan creation respects user's availability
The system SHALL skip dates the user marks as unavailable when generating tasks.

#### Scenario: Skip unavailable dates
- **WHEN** user marks certain dates as unavailable
- **THEN** AI does not create tasks on those dates

### Requirement: AI usage is controlled
The system SHALL enforce limits for AI generation requests to prevent accidental cost or abuse.

#### Scenario: User exceeds generation limit
- **WHEN** user exceeds the configured generation limit
- **THEN** system rejects new generation requests with a rate-limit message

#### Scenario: Default daily generation limit
- **WHEN** no custom AI usage limit is configured
- **THEN** system applies a default limit of 5 generation requests per user per day

### Requirement: AI provider is configurable
The system SHALL allow administrators to configure and validate the AI provider used by plan generation.

#### Scenario: Configure OpenAI-compatible provider
- **WHEN** admin configures base URL, model name, API key, timeout, and enabled state
- **THEN** system uses that provider for future AI generation requests

#### Scenario: Test provider connectivity
- **WHEN** admin runs an AI provider test from the PC admin console
- **THEN** system sends a minimal test request and reports whether provider access and response parsing succeeded

### Requirement: AI planning uses historical learning data
The system SHALL use relevant user history to adjust generated plan difficulty and pacing.

#### Scenario: User has low recent completion rate
- **WHEN** AI generates a plan for a user with low recent completion rate
- **THEN** system reduces daily load or increases schedule buffer in the generated plan

#### Scenario: User has strong recent completion rate
- **WHEN** AI generates a plan for a user with strong recent completion rate
- **THEN** system may generate a denser plan while staying within configured workload limits

### Requirement: AI agent can use controlled planning tools
The system SHALL allow the planning agent to use backend-owned allowlisted tools for user-specific learning context and schedule checks.

#### Scenario: Agent retrieves learning profile
- **WHEN** plan generation needs historical user context
- **THEN** agent may call an allowlisted backend tool that returns aggregated completion rate, average study minutes, streak, and postpone frequency for the authenticated user

#### Scenario: Agent checks schedule conflicts
- **WHEN** generated tasks include planned dates and times
- **THEN** agent may call an allowlisted backend tool to detect conflicts with the user's existing active plan load

#### Scenario: Tool access is restricted
- **WHEN** the model requests data outside the allowlisted planning tools
- **THEN** system refuses the tool request and does not expose raw database, credentials, SQL, or cross-user data

### Requirement: AI generation has fallback behavior
The system SHALL execute local planning as a first-class stage inside the asynchronous job and SHALL expose terminal failure only when no valid plan can be safely persisted.

#### Scenario: Provider is unavailable
- **WHEN** the external provider times out or fails after a valid local candidate exists
- **THEN** the worker persists the local candidate and marks the job succeeded with truthful fallback metadata

#### Scenario: No valid candidate can be persisted
- **WHEN** input, planning, final conflict checks, or persistence cannot produce a valid plan after bounded retries
- **THEN** the system marks the job failed with a safe actionable message and creates no partial plan or tasks

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
- **THEN** the planning Agent sends those preferences to the model as part of the decomposition brief and applies them where they do not conflict with authoritative constraints

#### Scenario: Model decomposition runs asynchronously
- **WHEN** a durable generation job invokes the configured model
- **THEN** the Agent uses the configured 5-minute or 10-minute background budget independently of the interactive request and connection-test timeout

#### Scenario: Additional instructions contradict constraints
- **WHEN** additional instructions request unavailable, conflicting, unauthorized, or otherwise invalid behavior
- **THEN** the system ignores or repairs the conflicting preference and persists only a validated plan
