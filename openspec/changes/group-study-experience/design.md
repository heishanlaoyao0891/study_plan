# Design: Group Study

## Invitations

Study groups support three joining methods:

- Invitation code typed manually.
- Mini program share link/payload.
- QR code or mini program code that carries the invitation scene.

These joining methods should not require payment. Generating mini program codes may be subject to WeChat API quotas and should be cached when possible. A code maps to a group and expires or can be revoked.

## Group Membership Boundary

A user SHALL have at most one active study group at a time. A user may join another group only after leaving the current group or after the current group has ended.

An MVP study group is not required to bind to one common study plan. Members may study their own plans, while the group shares only group-visible metrics such as check-ins, streaks, study minutes, completion rate, and level.

The group has a maximum of 10 active members in MVP.

The group name can be customized by the leader. Group avatar/image is out of scope for MVP.

Groups may have an `end_date` and may also be ended manually by the leader. If the leader wants to leave an active group, the leader must transfer leadership or end the group first.

Leadership can only be transferred to a current active member of the group.

Group roles:

- Leader: can remove members, end the group, and manage group invitation settings.
- Member: can invite others, view group dashboard, send nudges, and exit voluntarily.

## Member Status

Group members can see their own plans and the group-level status of other members. Members SHALL NOT see another member's private study plans or task details unless that member explicitly makes a plan/task public to the group.

Group-visible data includes member nickname/avatar, level, streak, group completion status, public study minutes, and completion rate. Private data includes non-public plans, non-public task descriptions, private notes, and non-group study details.

By default, group-visible metrics are public inside the group: continuous check-in days, study minutes, completion rate, level, and current-day completion state. Plan titles, task titles, task descriptions, and notes remain private unless the user explicitly marks them public to the group.

The public/private switch for plan/task details should live on the corresponding plan or task detail surface.

## Level System

Members earn levels based on check-ins, streaks, and sustained completion. The level system should be simple and transparent, for example total check-ins plus streak milestones. Levels are shown on member cards and leaderboard rows.

MVP level rules:

- Lv1: default.
- Lv2: continuous check-in streak >= 3 days.
- Lv3: continuous check-in streak >= 7 days.
- Lv4: continuous check-in streak >= 14 days.
- Lv5: continuous check-in streak >= 30 days.

## Leaderboard

Leaderboards are scoped to a study group and use group-visible metrics:

- Continuous check-in days.
- Study minutes.
- Completion rate.
- Member level.

MVP leaderboards include weekly ranking and all-time ranking.

## Nudges

Reminder nudges create notification events and are subject to WeChat subscription availability. The expected message is similar to: `xxx 提醒你赶紧开始学习了`. If the target user has not subscribed to the reminder template, the nudge is recorded but no WeChat message is sent.

Nudge limits:

- A member may nudge the same target member at most once per day.
- A member may receive at most 3 group nudges per day.

## Invitation Lifetime

Invitation codes and QR/miniprogram-code scene values expire after 7 days by default. The group leader may regenerate invitations.

## Leaving And History

When a user leaves a group, historical records remain for audit/statistics, but the user no longer appears in the active member leaderboard. Rejoining later creates or reactivates membership according to the one-active-group rule.

Ended groups should remain visible in a historical group detail view for former members.
