## ADDED Requirements

### Requirement: User receives reminder when study time arrives
The system SHALL send a WeChat subscription message at the scheduled start time of each task.

#### Scenario: Scheduled start reminder
- **WHEN** it is the task's scheduled_start time
- **THEN** system sends a message: "到学习时间了！[计划名] - [今日任务]"

### Requirement: User receives completion notification
The system SHALL notify when the daily scheduled study time is completed.

#### Scenario: Daily plan completed
- **WHEN** user's study duration reaches the scheduled_end
- **THEN** system sends: "今日计划学习已完成！共学习 X 小时"

### Requirement: 23:30 boundary decision reminder
The system SHALL remind users nearing midnight to decide whether to continue or postpone.

#### Scenario: Midnight boundary
- **WHEN** time is 23:30 and user has in_progress tasks
- **THEN** system sends: "已经23:30了，今日学习还未结束。是否延到明天继续？"

### Requirement: Missed checkin reminder
The system SHALL remind users who haven't started their scheduled study.

#### Scenario: No checkin reminder
- **WHEN** it is 30 minutes past scheduled_start and user has not started
- **THEN** system sends: "今日学习还没开始哦，记得来打卡～"

### Requirement: User can subscribe to reminders
The system SHALL allow users to subscribe to WeChat subscription messages.

#### Scenario: Subscribe
- **WHEN** user sends POST /api/notifications/subscribe with template ID
- **THEN** system records the subscription

#### Scenario: Unsubscribe
- **WHEN** user sends DELETE /api/notifications/subscribe
- **THEN** system removes the subscription record
