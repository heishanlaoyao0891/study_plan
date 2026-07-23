## MODIFIED Requirements

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

## ADDED Requirements

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
