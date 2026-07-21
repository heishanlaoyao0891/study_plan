## ADDED Requirements

### Requirement: Mini program exposes policy and support entries
The system SHALL expose privacy policy, user agreement, feedback, and version notes entries in the mini program.

#### Scenario: User opens settings
- **WHEN** user opens account/settings
- **THEN** user can access privacy policy, user agreement, feedback, and version notes

### Requirement: Admin can manage basic operations content
The system SHALL allow admins to manage simple announcements, feedback reports, and version notes from the PC admin console.

#### Scenario: Admin publishes announcement
- **WHEN** admin creates or updates an announcement
- **THEN** system makes the announcement available to mini program users according to display rules

#### Scenario: Admin reviews feedback
- **WHEN** admin opens feedback review
- **THEN** system shows submitted feedback/problem reports with timestamps and user context when available

### Requirement: Product copy follows release compliance guidance
The system SHALL avoid risky wording around AI, rankings, reminders, and slack minutes.

#### Scenario: AI copy displayed
- **WHEN** user sees AI-generated plan copy
- **THEN** system presents it as editable suggestion rather than guaranteed outcome or professional advice

#### Scenario: Slack copy displayed
- **WHEN** user sees slack minutes copy
- **THEN** system presents slack minutes as non-monetary rest/activity balance

### Requirement: Release checklist covers WeChat review risks
The system SHALL maintain a WeChat submission checklist covering privacy, phone usage, AI wording, slack wording, reminders, and account deletion.

#### Scenario: Prepare WeChat submission
- **WHEN** developer prepares a release
- **THEN** checklist confirms all compliance-sensitive entries are reviewed
