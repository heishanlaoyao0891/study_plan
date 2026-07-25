# plan-management Delta Specification

## ADDED Requirements

### Requirement: Generated plans reuse authoritative scheduling
The system SHALL use the same schedule, workload, and commit rules for Agent-generated plans as for user-created plans.

#### Scenario: Agent preview is committed
- **WHEN** the user accepts an Agent-generated preview
- **THEN** the backend revalidates current ownership, occupancy, workload, dates, time ranges, and derived totals transactionally before creating the plan

#### Scenario: Schedule changes after preview
- **WHEN** persisted tasks changed after the preview was generated
- **THEN** commit rejects or repairs stale scheduling instead of trusting the original client preview

### Requirement: Plan generation source is retained
The system SHALL distinguish manual, local Agent, and locally generated model-enriched plans.

#### Scenario: View plan provenance
- **WHEN** a generated plan is displayed
- **THEN** the product identifies whether it was locally planned or AI-enhanced without claiming all generated plans came from a model
