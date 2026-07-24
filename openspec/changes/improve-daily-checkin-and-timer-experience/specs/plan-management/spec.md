## MODIFIED Requirements

### Requirement: Each plan has individual time slots
The system SHALL allow each plan to define a default daily planned start/end range, selected study weekdays or explicit study dates, and optional day-specific time overrides.

#### Scenario: Set plan schedule
- **WHEN** user sets a default time, selected weekdays or dates, and optional overrides
- **THEN** system generates DailyTask entries only for selected study dates using a matching override first and default time otherwise

#### Scenario: Weekday or date is not selected
- **WHEN** a weekday or date is outside the plan's selected study schedule
- **THEN** the system does not generate a DailyTask for that day

#### Scenario: User creates a plan from the mini program
- **WHEN** the user opens the manual plan form
- **THEN** the UI provides default start/end controls, study-weekday or date selection, and optional schedule override editing

#### Scenario: No weekday override
- **WHEN** a selected study weekday has no explicit override
- **THEN** generated tasks use the plan's default planned start/end range

#### Scenario: Weekday override exists
- **WHEN** a selected study weekday has an explicit override
- **THEN** generated tasks use that weekday's overridden planned start/end range

### Requirement: User can edit a study plan
The system SHALL allow users to update plan properties and schedule templates without silently changing already-generated tasks.

#### Scenario: Edit a plan
- **WHEN** user updates title, dates, default schedule, study weekdays, or weekday overrides
- **THEN** system saves the plan template for future task generation

#### Scenario: Existing tasks after schedule edit
- **WHEN** a schedule template changes and tasks already exist
- **THEN** existing tasks preserve their planned times unless the user explicitly selects regeneration or batch update

## ADDED Requirements

### Requirement: Planned time does not restrict actual learning
The system SHALL use planned task times for target duration, schedule display, and reminders without preventing flexible actual start and pause behavior.

#### Scenario: User starts outside scheduled time
- **WHEN** user starts a generated task outside its planned clock range
- **THEN** system permits the session and retains the original planned range for reporting and target duration
