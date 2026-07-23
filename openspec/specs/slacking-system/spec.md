# slacking-system Specification

## Purpose
TBD - created by archiving change study-checkin-miniapp. Update Purpose after archive.
## Requirements
### Requirement: User earns slack time through study
The system SHALL accumulate slack minutes based on study performance and configurable rules.

#### Scenario: Earn from daily checkin
- **WHEN** user completes a daily checkin
- **THEN** system adds configured checkin_minutes to slack_balance

#### Scenario: Earn from continuous streak
- **WHEN** user achieves N consecutive perfect days
- **THEN** system awards streak_bonus minutes

#### Scenario: Earn from quality completion
- **WHEN** user completes all tasks for N consecutive days
- **THEN** system awards quality_bonus minutes

### Requirement: User can start slacking
The system SHALL allow users to start a slack session, consuming their slack balance.

#### Scenario: Start slacking
- **WHEN** user taps "开始躺平"
- **THEN** system prompts to enter activity description (e.g. "刷视频", "钓鱼")
- **WHEN** user confirms and enters activity
- **THEN** system records slack start_time and deducts from balance in real-time

### Requirement: User can end slacking
The system SHALL record the end of a slack session and calculate duration.

#### Scenario: End slacking
- **WHEN** user taps "结束躺平"
- **THEN** system records end_time, calculates duration_min, deducts from balance

### Requirement: Slack activity must be logged
The system SHALL require users to describe what they did during slack time.

#### Scenario: Log activity
- **WHEN** user starts slacking
- **THEN** system requires an activity description before starting

### Requirement: Slack balance is visible
The system SHALL display current slack balance to the user.

#### Scenario: View balance
- **WHEN** user checks slack balance
- **THEN** system shows total available slack minutes

### Requirement: User can view slack history
The system SHALL show a log of past slack sessions.

#### Scenario: View slack records
- **WHEN** user requests GET /api/slack/records
- **THEN** system returns past slack sessions with activity, duration, and date

### Requirement: Slack activity distribution is visualizable
The system SHALL show a breakdown of how slack time was spent.

#### Scenario: Slack distribution
- **WHEN** user requests GET /api/stats/slack-distribution?month=2026-07
- **THEN** system returns activities grouped by type with total duration

### Requirement: Makeup study time consumes slack minutes
The system SHALL consume slack minutes when users manually make up study time, using a configurable ratio.

#### Scenario: Makeup accepted with sufficient slack
- **WHEN** user submits makeup study time and has sufficient slack balance
- **THEN** system deducts slack minutes according to the configured ratio and records the deduction

#### Scenario: Makeup rejected with insufficient slack
- **WHEN** user submits makeup study time but lacks required slack balance
- **THEN** system rejects the makeup request under MVP policy

### Requirement: Suspicious study records are flagged
The system SHALL mark abnormal study records for admin review without automatically banning users.

#### Scenario: Excessive makeup duration
- **WHEN** user submits an unusually long makeup or study record
- **THEN** system marks the record suspicious for admin visibility

### Requirement: Slack minutes have no monetary value
The system SHALL present slack minutes only as rest/activity balance and not as money or redeemable value.

#### Scenario: User views slack page
- **WHEN** user opens slack page
- **THEN** system describes slack minutes as rest/activity balance without money-related wording

### Requirement: User receives rest balance suggestions
The system SHALL provide simple study/rest balance suggestions based on study and slack history.

#### Scenario: High study week
- **WHEN** user has high study minutes and available slack balance
- **THEN** system may suggest a reasonable rest activity or rest duration

