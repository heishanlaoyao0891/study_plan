## ADDED Requirements

### Requirement: User can have multiple study plans
The system SHALL support multiple concurrent study plans for a single user.

#### Scenario: Create multiple plans
- **WHEN** user creates a second study plan (e.g. "学习 English")
- **THEN** system manages both plans independently with their own tasks and schedules

### Requirement: Each plan has a weekly target hours
The system SHALL track weekly study hours per plan and alert when below target.

#### Scenario: Set weekly target
- **WHEN** user creates or edits a plan with weekly_target_hours (e.g. 28h)
- **THEN** system tracks progress against this target and shows completion rate

#### Scenario: Below target warning
- **WHEN** end of week and total study hours < weekly_target_hours
- **THEN** system records as incomplete week

### Requirement: Each plan has individual time slots
The system SHALL allow each plan to have its own scheduled time slots.

#### Scenario: Set plan schedule
- **WHEN** user sets "Go" plan to Mon-Fri 20:00-22:00
- **THEN** system generates DailyTask entries with those time slots for each study day

### Requirement: Plan can be paused and resumed
The system SHALL allow users to temporarily freeze a plan without losing data.

#### Scenario: Pause a plan
- **WHEN** user sends PUT /api/plans/:id/pause
- **THEN** system sets plan status to "paused" and stops generating new tasks

#### Scenario: Resume a plan
- **WHEN** user sends PUT /api/plans/:id/resume
- **THEN** system sets plan status to "active" and continues task generation

### Requirement: Plan can be shifted entirely
The system SHALL allow shifting all future tasks of a plan by a number of days.

#### Scenario: Shift all tasks
- **WHEN** user sends PUT /api/plans/:id/shift with days=3
- **THEN** system moves all future tasks by 3 days

### Requirement: User can view their study plans
The system SHALL return all plans for the authenticated user.

#### Scenario: List all plans
- **WHEN** user requests GET /api/plans
- **THEN** system returns an array of plans sorted by user-defined order

### Requirement: User can create a study plan manually
The system SHALL allow users to create a new study plan item without AI.

#### Scenario: Create a plan manually
- **WHEN** user sends POST /api/plans with title and schedule
- **THEN** system creates the plan and user can manually add daily tasks

### Requirement: System warns when plans overlap or overload
The system SHALL check existing active plans and warn the user if the new plan would cause excessive workload.

#### Scenario: Time slot overlap warning
- **WHEN** user creates a plan with time slot 20:00-22:00 and an existing active plan already occupies 20:00-21:00
- **THEN** system returns a warning: "该时间段与 [计划名] 重叠，是否确认？"

#### Scenario: Total weekly hours overload warning
- **WHEN** user's new plan would bring total weekly hours across all active plans above a threshold (e.g. 56h/week)
- **THEN** system returns a warning: "所有计划每周总学时已达 X 小时，压力可能过大，是否确认？"

#### Scenario: Too many active plans warning
- **WHEN** user already has 3+ active plans and tries to create another
- **THEN** system returns a warning: "已有 X 个活跃计划，不建议同时进行过多学习计划"

### Requirement: User can override overload warnings
The system SHALL allow the user to confirm and proceed despite warnings.

#### Scenario: Confirm override
- **WHEN** system shows overload warning and user chooses to proceed anyway
- **THEN** system creates the plan without further阻拦

### Requirement: Plan can be shared with other users
The system SHALL allow a plan to be shared among multiple users for collaborative learning.

#### Scenario: Create shared plan
- **WHEN** user creates a plan and adds other users as members
- **THEN** all members can view the plan, its tasks, and each other's progress

#### Scenario: Join shared plan
- **WHEN** user B accepts an invitation to join user A's plan
- **THEN** B becomes a member and can see and check in on the shared plan

### Requirement: Shared plan shows all members' status
The system SHALL display each member's task completion for shared plans.

#### Scenario: View member progress
- **WHEN** any member views the shared plan
- **THEN** system shows each member's daily completion status

### Requirement: User can edit a study plan
The system SHALL allow users to update an existing plan's properties.

#### Scenario: Edit a plan
- **WHEN** user sends PUT /api/plans/:id with new title or schedule
- **THEN** system updates the plan

### Requirement: User can delete a study plan
The system SHALL allow users to delete a plan and its associated records.

#### Scenario: Delete a plan
- **WHEN** user sends DELETE /api/plans/:id
- **THEN** system deletes the plan and all related tasks, checkins, and records
