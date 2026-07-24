## ADDED Requirements

### Requirement: User receives one concise daily motivation
The system SHALL provide one stable motivational message per authenticated user per Asia/Shanghai calendar day.

#### Scenario: First visit of the day
- **WHEN** the user opens the daily check-in page and has no cached message for the current date
- **THEN** the system generates or selects one message and stores it for that user/date

#### Scenario: Repeated visit on same day
- **WHEN** the user reopens or refreshes the check-in page on the same date
- **THEN** the system returns the same cached message

### Requirement: Daily motivation uses safe AI fallback
The system SHALL attempt configured AI generation and SHALL use a moderated built-in quote when AI is disabled, unavailable, times out, or returns invalid content.

#### Scenario: AI generates valid original text
- **WHEN** AI returns valid motivational text
- **THEN** the system labels its source as `今日寄语` and does not assign a fabricated author

#### Scenario: AI generation fails
- **WHEN** AI is unavailable or its output fails validation
- **THEN** the system returns and caches a moderated fallback quote with a verified source

### Requirement: Daily motivation fits the check-in layout
The system SHALL enforce a maximum of 32 Chinese-display characters for message text, 12 characters for source, and a maximum rendered height of two text lines.

#### Scenario: AI returns over-limit content
- **WHEN** generated text or source exceeds the configured display limit
- **THEN** the system rejects that output and uses valid fallback content instead of truncating it in the client

#### Scenario: Motivation card renders on a small screen
- **WHEN** the check-in page is displayed on a supported small viewport
- **THEN** the motivation card preserves its fixed two-line content area without pushing primary task actions out of the intended first-screen layout

### Requirement: Motivation personalization protects user privacy
The system SHALL limit AI personalization context to aggregate learning signals and SHALL NOT expose private task text or cross-user data.

#### Scenario: Generate personalized encouragement
- **WHEN** the system prepares the daily motivation prompt
- **THEN** it may include streak and recent completion-rate summaries but excludes task descriptions, private notes, credentials, and other users' data
