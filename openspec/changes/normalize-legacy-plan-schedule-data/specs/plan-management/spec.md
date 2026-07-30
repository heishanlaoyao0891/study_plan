## MODIFIED Requirements

### Requirement: User can view their study plans
The system SHALL return all plans for the authenticated user with stable array-shaped schedule fields, including `study_weekdays`, `study_dates`, and schedule overrides, even when a historical record has no stored schedule array.

#### Scenario: List all plans
- **WHEN** user requests GET /api/plans
- **THEN** system returns an array of plans sorted by user-defined order

#### Scenario: List legacy plan with missing schedule arrays
- **WHEN** a user requests plans containing a historical record whose schedule arrays are absent
- **THEN** the system returns empty arrays rather than JSON null for the missing schedule fields

#### Scenario: Render legacy plan schedule
- **WHEN** the mini-program receives a plan whose schedule fields are null from a stale or nonconforming source
- **THEN** the plan list and detail views render a safe unscheduled state without throwing an error
