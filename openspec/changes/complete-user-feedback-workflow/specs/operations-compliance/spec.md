# operations-compliance Delta Specification

## MODIFIED Requirements

### Requirement: Mini program exposes policy and support entries
The system SHALL expose a discoverable authenticated feedback and problem-report workflow in both H5 and the mini program, alongside privacy policy, user agreement, and version notes.

#### Scenario: User opens support entry
- **WHEN** an authenticated user opens account/settings or settings-and-policy
- **THEN** the user can navigate to a dedicated feedback page without searching through policy content

#### Scenario: User submits feedback
- **WHEN** the user selects an allowed category, enters valid content, and submits once
- **THEN** the system stores the report for that account and shows a clear success state without creating a duplicate submission

#### Scenario: User reviews submitted reports
- **WHEN** the user opens feedback history
- **THEN** the system shows only that user's reports with category, content, status, public administrator response, and update time

### Requirement: Admin can manage basic operations content
The system SHALL allow administrators to triage feedback reports, provide a user-visible response, and move each report through the supported lifecycle.

#### Scenario: Admin filters feedback
- **WHEN** an administrator filters by category or status
- **THEN** the inbox shows matching reports with timestamp and safe user context

#### Scenario: Admin handles feedback
- **WHEN** an administrator saves a response or changes a report status
- **THEN** the system records the administrator and response time and exposes the public response to only the report owner

#### Scenario: Non-admin attempts handling
- **WHEN** a non-administrator calls a feedback handling endpoint
- **THEN** the system rejects the request without changing the report

## ADDED Requirements

### Requirement: Feedback input is bounded and abuse resistant
The system SHALL validate feedback categories and field lengths on the server and apply authenticated submission throttling.

#### Scenario: Invalid feedback is submitted
- **WHEN** category, content, or contact violates the feedback contract
- **THEN** the system rejects the report with a field-specific correction message

#### Scenario: Feedback is submitted repeatedly
- **WHEN** one account exceeds the supported short-window submission rate
- **THEN** the system rejects additional reports temporarily without creating rows
