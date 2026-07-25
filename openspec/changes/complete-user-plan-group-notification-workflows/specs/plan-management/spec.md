# plan-management Delta Specification

## ADDED Requirements

### Requirement: Manual plan creation uses per-date task drafts
The system SHALL let users review and edit a concrete task for every selected study date before creating a scheduled manual plan.

#### Scenario: Generate task drafts
- **WHEN** the user chooses a date range and study weekdays
- **THEN** the client generates rows only for matching dates and allows each row to have a different title, objective, description, and time

#### Scenario: Create explicit tasks
- **WHEN** valid task drafts are submitted
- **THEN** the backend creates the plan and exactly those tasks atomically

### Requirement: Plan detail manages plan and tasks
The system SHALL open plan detail from the plan card and expose owned daily tasks with create, edit, and delete controls.

#### Scenario: Open plan detail
- **WHEN** the user taps a plan card
- **THEN** the system shows plan properties, invitation, lifecycle controls, progress, and date-ordered task details

### Requirement: Empty today explains next work
The system SHALL distinguish no task today from missing plan tasks.

#### Scenario: Today has no task
- **WHEN** check-in has no task on the requested date but a future pending task exists
- **THEN** the page shows that next task's date/title and a plan-detail link without treating it as today's work

## MODIFIED Requirements

### Requirement: Study group has leader and member permissions
The system SHALL allow every active member to generate invitations and SHALL let the leader remove members with confirmation.

#### Scenario: Member invites another user
- **WHEN** an active group member requests an invitation
- **THEN** the system returns a valid seven-day invitation

### Requirement: Shared plan has a leaderboard
The system SHALL calculate weekly scope from current-week persisted activity rather than all-time aggregates.

#### Scenario: View weekly leaderboard
- **WHEN** a member requests the weekly leaderboard
- **THEN** ranking uses only the current Asia/Shanghai week
