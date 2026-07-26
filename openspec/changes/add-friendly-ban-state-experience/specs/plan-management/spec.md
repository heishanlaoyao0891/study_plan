## MODIFIED Requirements

### Requirement: Plan tasks can be shifted in batch
The system SHALL present an editable, non-persisted preview before delaying eligible future unfinished tasks and SHALL validate the complete resulting schedule before applying it atomically.

#### Scenario: User previews a plan delay
- **WHEN** the user selects a positive delay
- **THEN** the client opens a preview showing every eligible task's current and proposed date and time
- **AND** no plan or task is changed

#### Scenario: Proposed delay conflicts with existing work
- **WHEN** a proposed task reaches the planned-overlap limit or uses a date already occupied by another task in the same plan
- **THEN** the preview identifies the conflict and disables confirmation
- **AND** the user can edit the proposed date and time

#### Scenario: User applies an edited delay
- **WHEN** every proposed task is valid and the preview is still current
- **THEN** the backend revalidates ownership, versions, states, unique plan dates, and schedule overlap
- **AND** moves all previewed tasks and updates the plan range in one transaction

#### Scenario: Consecutive tasks move forward
- **WHEN** consecutive task dates are delayed into dates currently occupied by other tasks in the same moving set
- **THEN** the system avoids transient unique-index conflicts and persists the valid final schedule

#### Scenario: Delay preview becomes stale
- **WHEN** a previewed task changes before apply
- **THEN** the backend rejects the apply as stale without moving any task
