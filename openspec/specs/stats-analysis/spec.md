# stats-analysis Specification

## Purpose
TBD - created by archiving change study-checkin-miniapp. Update Purpose after archive.
## Requirements
### Requirement: User can view monthly calendar stats
The system SHALL return daily study data for a given month.

#### Scenario: View monthly stats
- **WHEN** user requests GET /api/stats/calendar?month=2026-07
- **THEN** system returns each day with: total_study_minutes, completed_tasks, total_tasks, is_completed

### Requirement: User can view current streak
The system SHALL calculate consecutive days of full completion.

#### Scenario: View streak
- **WHEN** user requests GET /api/stats/streak
- **THEN** system returns current consecutive fully-completed days count

### Requirement: A day counts as complete only when all plans are done
The system SHALL consider a day "fully completed" only when every plan has been checked in.

#### Scenario: Partial completion
- **WHEN** user has completed 2 out of 3 plans on a day
- **THEN** that day is NOT counted toward the streak

### Requirement: User can view daily study time distribution
The system SHALL show a breakdown of study time by plan for a given day.

#### Scenario: Daily distribution
- **WHEN** user requests GET /api/stats/daily-distribution?date=2026-07-17
- **THEN** system returns each plan with study_minutes and checkin status

### Requirement: User can view weekly report
The system SHALL generate an automatic weekly study summary.

#### Scenario: Weekly report
- **WHEN** user requests GET /api/stats/weekly-report?year=2026&week=29
- **THEN** system returns: total_study_min, target_min, completed_rate, slack_min, perfect_days

### Requirement: User can view monthly report
The system SHALL generate a monthly study summary.

#### Scenario: Monthly report
- **WHEN** user requests GET /api/stats/monthly-report?year=2026&month=7
- **THEN** system returns: total_study_hours, completion_rate, slack_hours, longest_streak, weekly_breakdown

### Requirement: User can view slack activity distribution
The system SHALL show what activities slack time was spent on.

#### Scenario: Slack distribution
- **WHEN** user requests GET /api/stats/slack-distribution?month=2026-07
- **THEN** system returns activities grouped by type with total duration

### Requirement: Overtime study is included in stats
The system SHALL count overtime study minutes in all statistical calculations.

#### Scenario: Overtime counted in stats
- **WHEN** user studies overtime on any day
- **THEN** overtime minutes are included in daily/weekly/monthly totals

### Requirement: System analyzes personal learning efficiency
The system SHALL track study efficiency metrics over time.

#### Scenario: Efficiency analysis
- **WHEN** user has sufficient historical data
- **THEN** system provides metrics like: avg daily study time, task completion rate, plan progress velocity, weekly target hit rate

