## ADDED Requirements

### Requirement: Active study session can be recovered across authenticated clients
The system SHALL expose the authenticated user's currently open study session as its timer task view, including the original session start time and accumulated duration. The system SHALL return no active task when the user has no open study session and SHALL never expose another user's session.

#### Scenario: Recover a task started on another client
- **WHEN** a user starts a task in the mini program and then authenticates in H5 while the session remains open
- **THEN** H5 receives the same active task with elapsed duration calculated from the original server-side session start time

#### Scenario: No active task
- **WHEN** an authenticated user has no open study session
- **THEN** the active-task query returns an empty result and the client uses its normal post-login route

#### Scenario: Active task remains private
- **WHEN** a user requests the active-task query
- **THEN** the system returns only an open session owned by that authenticated user

### Requirement: Authentication restores an active study task
The client SHALL, after successfully storing a complete account's authentication token, check for an active study task before navigating to the normal check-in route. If one exists, the client SHALL open its task detail without sending a start or resume mutation.

#### Scenario: H5 login resumes a running timer
- **WHEN** H5 login succeeds for a user with an active study session
- **THEN** the client opens the active task and displays the continuing server-derived duration

#### Scenario: WeChat login resumes a running timer
- **WHEN** mini-program authentication succeeds for a user with an active study session
- **THEN** the client opens the active task and displays the continuing server-derived duration

### Requirement: Recovered task detail has a stable exit to the task list
The task detail view SHALL provide a persistent navigation action to the check-in task list. The action SHALL work when the task detail was opened through authentication recovery and SHALL not pause, resume, stop, or create a study session.

#### Scenario: Leave a recovered task detail view
- **WHEN** a user opens task detail through H5 or mini-program authentication recovery and selects the task-list action
- **THEN** the client navigates to the check-in task list while the active session remains unchanged
