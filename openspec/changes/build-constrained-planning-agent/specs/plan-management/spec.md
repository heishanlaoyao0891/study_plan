# plan-management Delta Specification

## ADDED Requirements

### Requirement: Generated plans reuse authoritative scheduling
The system SHALL apply the same schedule, workload, ownership, and commit rules to local and AI-decomposed previews as to user-created plans.

#### Scenario: Schedule an AI task blueprint
- **WHEN** a model job returns ordered semantic tasks and estimated effort
- **THEN** the backend assigns eligible dates and time ranges, preserves prerequisite order, repairs conflicts, and validates the resulting preview before publication

#### Scenario: Agent preview is committed
- **WHEN** the user accepts a local or AI-decomposed preview version
- **THEN** the backend revalidates current ownership, occupancy, workload, dates, time ranges, and derived totals transactionally before creating the plan

#### Scenario: Schedule changes after preview
- **WHEN** persisted tasks changed after a preview version was generated
- **THEN** commit rejects or repairs stale scheduling instead of trusting the client preview

#### Scenario: AI task exceeds one daily slot
- **WHEN** an AI-decomposed task is larger than the available daily capacity
- **THEN** the Agent creates ordered task parts with preserved intent and returns a repair warning

### Requirement: Generated preview versions are immutable
The system SHALL keep local and AI-decomposed preview versions independently addressable until expiry or commit.

#### Scenario: AI version arrives after local edits
- **WHEN** the user edits a local version before an AI-decomposed version is published
- **THEN** both versions remain available for review and neither silently mutates the other

#### Scenario: One version is committed
- **WHEN** the user commits a valid preview version
- **THEN** the created plan records that version's normalized content and later job updates cannot change it

#### Scenario: User restructures preview tasks
- **WHEN** the user adds, removes, splits, or reorders tasks in a generated preview
- **THEN** the backend creates and validates a new derived preview version rather than accepting client-invented task identities

### Requirement: Plan generation source is retained
The system SHALL distinguish manual, local Agent, and AI-decomposed plans without claiming that fallback plans came from a model.

#### Scenario: View plan provenance
- **WHEN** a generated plan is displayed
- **THEN** the product identifies its source as `manual`, `local`, or `ai_decomposed`

#### Scenario: Local baseline is committed while AI runs
- **WHEN** the user commits before model decomposition completes
- **THEN** the stored plan source remains `local` even if the job later produces an AI version

#### Scenario: AI decomposition has not produced valid output
- **WHEN** the current provider attempt times out or returns invalid output
- **THEN** no plan or preview is marked `ai_decomposed` and the AI job remains retryable
