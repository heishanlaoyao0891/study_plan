## MODIFIED Requirements

### Requirement: User can start learning on schedule
The system SHALL allow a user to start a task at any time only when that user has no different active study task.

#### Scenario: Start with no active task
- **WHEN** the user starts an incomplete task and has no open StudySession
- **THEN** the system creates an active session and sets that task to in progress

#### Scenario: Repeat start for the active task
- **WHEN** the user repeats start or resume for the task that already owns their active session
- **THEN** the system returns the existing session without creating another session

#### Scenario: Start while another task is active
- **WHEN** the user starts or resumes task B while task A has an open StudySession
- **THEN** the backend returns HTTP 409 with task A's ID, title, and active session ID
- **AND** neither task A nor task B is paused, completed, or otherwise mutated

### Requirement: User can pause and resume learning
The system SHALL allow pause and resume while preserving accumulated sessions and SHALL enforce one active study task per user when resuming.

#### Scenario: Resume after pausing the same task
- **WHEN** the user has no active session and resumes a paused task
- **THEN** the system creates a new StudySession for that task

#### Scenario: Resume while another task is active
- **WHEN** the user attempts to resume a paused task while a different task is active
- **THEN** the system rejects the resume and identifies the active task

## ADDED Requirements

### Requirement: A user has only one active study task
The system SHALL enforce at most one open StudySession per user across all plans and tasks.

#### Scenario: Concurrent starts for different tasks
- **WHEN** start requests for two different tasks belonging to the same user arrive concurrently
- **THEN** exactly one task obtains an active session
- **AND** the other request returns a task-conflict response

#### Scenario: Different users study concurrently
- **WHEN** two different users start their own tasks
- **THEN** each user may have one active session independently

#### Scenario: Existing active task appears on daily page
- **WHEN** the daily page contains one running task and other incomplete tasks
- **THEN** the running task keeps its pause or completion controls
- **AND** other tasks cannot be started or resumed until the active task is paused or completed

#### Scenario: App state is stale
- **WHEN** the client attempts to start a task based on stale state and the backend reports another active task
- **THEN** the client refreshes authoritative timer state and explains which task is currently learning
