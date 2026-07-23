## MODIFIED Requirements

### Requirement: User can create a study plan manually
The system SHALL allow users to create a new study plan manually without leaving them in an empty, unclear state.

#### Scenario: Create a plan manually
- **WHEN** user sends POST /api/plans with title and schedule
- **THEN** system creates the plan and user can manually add daily tasks

#### Scenario: Create a manual plan with schedule
- **WHEN** user sends POST /api/plans with title and schedule information
- **THEN** system creates the plan and generates daily tasks for the scheduled date range

#### Scenario: Create a manual plan without schedule
- **WHEN** user creates a plan without enough schedule information to generate tasks
- **THEN** the client clearly guides the user to add tasks or complete schedule settings before expecting today's check-in items

### Requirement: Each plan has individual time slots
The system SHALL allow each plan to have its own scheduled time slots and present those schedule choices clearly during manual creation.

#### Scenario: Set plan schedule
- **WHEN** user sets "Go" plan to Mon-Fri 20:00-22:00
- **THEN** system generates DailyTask entries with those time slots for each study day

#### Scenario: User creates a plan from the mini program
- **WHEN** the user opens the manual plan form
- **THEN** the UI provides understandable fields for date range and planned time slot or explicitly labels the plan as taskless until tasks are added

## ADDED Requirements

### Requirement: Mini program plan experience feels warm and motivating
The system SHALL present plan creation and plan-list views with a cute, motivating visual style suitable for daily learning.

#### Scenario: User views plan list
- **WHEN** the user opens the plan tab
- **THEN** plan cards, status labels, and primary actions use the shared cute visual system
- **AND** secondary actions do not overwhelm the card layout
