# plan-management Specification

## Purpose
TBD - created by archiving change study-checkin-miniapp. Update Purpose after archive.
## Requirements
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
The system SHALL allow each plan to have its own scheduled time slots and present those schedule choices clearly during manual creation.

#### Scenario: Set plan schedule
- **WHEN** user sets "Go" plan to Mon-Fri 20:00-22:00
- **THEN** system generates DailyTask entries with those time slots for each study day

#### Scenario: User creates a plan from the mini program
- **WHEN** the user opens the manual plan form
- **THEN** the UI provides understandable fields for date range and planned time slot or explicitly labels the plan as taskless until tasks are added

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
The system SHALL allow a study group to be joined using an invitation code, mini program share payload, or QR/miniprogram code.

#### Scenario: Create shared plan
- **WHEN** user creates a plan and adds other users as members
- **THEN** all members can view the plan, its tasks, and each other's progress

#### Scenario: Join shared plan
- **WHEN** user B accepts an invitation to join user A's plan
- **THEN** B becomes a member and can see and check in on the shared plan

#### Scenario: Create invitation code
- **WHEN** group leader or member requests an invitation for a study group
- **THEN** system creates a join code, share payload, or QR/miniprogram code tied to that group

#### Scenario: Join by invitation code
- **WHEN** another authenticated user submits a valid invitation code
- **THEN** system adds that user as a member of the study group if the user is not already in another active group

#### Scenario: Join by share or QR code
- **WHEN** another authenticated user opens a valid group share payload or QR/miniprogram code
- **THEN** system lets that user join the study group if the user is not already in another active group

#### Scenario: Invitation expires
- **WHEN** user submits an invitation older than 7 days
- **THEN** system rejects the invitation as expired

### Requirement: Shared plan shows all members' status
The system SHALL display group-visible member status, study minutes, level, completion rate, and check-in state for study groups while protecting private plans and tasks.

#### Scenario: View member progress
- **WHEN** any member views the shared plan
- **THEN** system shows each member's daily completion status

#### Scenario: View group dashboard
- **WHEN** a member opens a group dashboard
- **THEN** system shows every member's current-day group status without exposing private plan/task details

#### Scenario: Private plan details hidden
- **WHEN** member A views member B in the group dashboard
- **THEN** system hides member B's plan titles, task titles, task descriptions, and private notes unless member B explicitly made them public to the group

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

### Requirement: Shared plan has a leaderboard
The system SHALL provide a leaderboard for members of a study group.

#### Scenario: View shared plan leaderboard
- **WHEN** a member opens the leaderboard
- **THEN** system ranks members by configured metrics such as continuous check-in days, study minutes, completion rate, and level

#### Scenario: View weekly leaderboard
- **WHEN** a member opens the weekly leaderboard
- **THEN** system ranks active members using current-week group-visible metrics

#### Scenario: View all-time leaderboard
- **WHEN** a member opens the all-time leaderboard
- **THEN** system ranks active members using all-time group-visible metrics

### Requirement: Members can nudge each other
The system SHALL allow study group members to send reminder nudges to other members through WeChat subscription messages when permitted.

#### Scenario: Send reminder nudge
- **WHEN** member A sends a nudge to member B in the same study group
- **THEN** system creates a notification event for member B and sends a WeChat reminder if member B has subscribed to the reminder template

#### Scenario: Nudge target has not subscribed
- **WHEN** member A nudges member B but member B has not subscribed to the reminder template
- **THEN** system records the nudge attempt without sending a WeChat message

#### Scenario: Same target nudge limit reached
- **WHEN** member A has already nudged member B today
- **THEN** system rejects another nudge from member A to member B on the same day

#### Scenario: Receive nudge limit reached
- **WHEN** member B has already received 3 group nudges today
- **THEN** system rejects additional group nudges to member B that day

### Requirement: User can join only one active study group
The system SHALL prevent a user from joining more than one active study group at the same time.

#### Scenario: User already in active group
- **WHEN** user tries to join another group while already in an active group
- **THEN** system rejects the join request with a clear message

#### Scenario: Current group ended
- **WHEN** user's current study group has ended or the user has left it
- **THEN** system allows the user to join another active group

#### Scenario: Group member limit reached
- **WHEN** user tries to join a group that already has 10 active members
- **THEN** system rejects the join request with a group-full message

### Requirement: Study group has leader and member permissions
The system SHALL enforce group permissions for leader and member actions.

#### Scenario: Leader removes member
- **WHEN** group leader removes a member
- **THEN** system removes that member from the group

#### Scenario: Member invites another user
- **WHEN** group member creates or shares an invitation
- **THEN** system allows the invitation if the group is active

#### Scenario: Member exits group
- **WHEN** group member chooses to exit
- **THEN** system removes that member from the group

#### Scenario: Leader tries to exit active group
- **WHEN** group leader tries to exit an active group without transferring leadership or ending the group
- **THEN** system rejects the exit request

#### Scenario: Member tries to remove another member
- **WHEN** non-leader member tries to remove another member
- **THEN** system rejects the action

### Requirement: Member level reflects check-in performance
The system SHALL calculate a member level based on check-ins and streak milestones.

#### Scenario: Member streak increases
- **WHEN** member completes check-ins and increases continuous days
- **THEN** system updates or displays the member's level according to configured level rules

#### Scenario: Level thresholds applied
- **WHEN** member has continuous check-in streak of 3, 7, 14, or 30 days
- **THEN** system displays Lv2, Lv3, Lv4, or Lv5 respectively

### Requirement: Study group does not require a common plan
The system SHALL allow members of a study group to study their own plans while sharing only group-visible metrics.

#### Scenario: Member studies private plan
- **WHEN** member completes study tasks from a private personal plan
- **THEN** system may count group-visible metrics such as study minutes and completion state without exposing the private plan details

### Requirement: Leaving group preserves history
The system SHALL preserve historical group records when a member leaves, while excluding the former member from active member leaderboards.

#### Scenario: Member leaves group
- **WHEN** member leaves a group
- **THEN** system preserves historical records but removes the member from active member leaderboard views

### Requirement: Mini program plan experience feels warm and motivating
The system SHALL present plan creation and plan-list views with a cute, motivating visual style suitable for daily learning.

#### Scenario: User views plan list
- **WHEN** the user opens the plan tab
- **THEN** plan cards, status labels, and primary actions use the shared cute visual system
- **AND** secondary actions do not overwhelm the card layout

### Requirement: Personal utilities are separated from plan management
The client SHALL keep account/data and settings/help entries out of the plan page and SHALL place them in a utility section at the bottom of the rightmost statistics tab.

#### Scenario: User opens plan tab
- **WHEN** a user views the plan list
- **THEN** the page contains plan creation, scheduling, group, reminder, recovery, and plan content without `账号与数据` or `设置与说明` entries

#### Scenario: User reaches the bottom of statistics
- **WHEN** a user scrolls past all statistics content in the rightmost tab
- **THEN** a separated utility section shows `账号与数据` followed by `设置与说明`

#### Scenario: User opens relocated utility
- **WHEN** a user selects either relocated entry
- **THEN** the client opens the existing account/data or settings/help destination without changing its underlying behavior
