# plan-management Delta Specification

## MODIFIED Requirements

### Requirement: User can edit a study plan
The system SHALL allow users to update plan properties and schedule templates from plan detail, SHALL preserve date-specific overrides not edited by that interface, and SHALL clearly disclose effects on existing tasks before applying a schedule change.

#### Scenario: Edit default plan schedule
- **WHEN** the owner changes the default planned start/end range or selected study weekdays in plan detail
- **THEN** the client validates a selected weekday or pre-existing explicit study date and a positive time range before submitting the supported plan update fields

#### Scenario: Edit plan copy without changing schedule
- **WHEN** the owner changes only title, description, or weekly target
- **THEN** the client omits schedule fields from the update request
- **AND** pending, running, and completed task times remain unchanged

#### Scenario: Edit weekday override
- **WHEN** the owner enables a custom range for a selected weekday
- **THEN** the client submits that weekday override after validating its planned end is later than its planned start

#### Scenario: Preserve date override
- **WHEN** a plan contains an existing date-specific override and the owner edits default or weekday schedule fields
- **THEN** the client includes the unchanged date override in the update payload
- **AND** the backend continues resolving it before weekday overrides and defaults

#### Scenario: Schedule change affects existing tasks
- **WHEN** schedule fields differ from the loaded plan and pending or running tasks exist
- **THEN** the client explains that those non-completed tasks will receive resolved template times and requests explicit confirmation
- **AND** explains that completed task times remain unchanged

#### Scenario: Schedule mutation is invalid
- **WHEN** client or backend schedule validation rejects the resulting ranges or overlap
- **THEN** no partial plan or task schedule update is committed
- **AND** the editor remains available for correction

### Requirement: Plan detail actions use clear hierarchy
The mini program SHALL use a fixed plan-detail action hierarchy without configurable or draggable action placement.

#### Scenario: View active plan operations
- **WHEN** an owner views an active or paused plan
- **THEN** pause or resume is the visually primary lifecycle action
- **AND** edit, invite, and more are visually secondary

#### Scenario: Change lifecycle status
- **WHEN** the owner taps pause or resume
- **THEN** the client explains the resulting lifecycle state and requests confirmation before mutation

#### Scenario: View destructive action
- **WHEN** the owner opens more actions
- **THEN** delete appears as a muted destructive action separated from routine operations
- **AND** final deletion still requires confirmation explaining affected tasks and check-in history
