# Design: Anti-Abuse And Slack Balance

## Makeup Cost

When users manually make up study time, the system should consume slack minutes at a configurable ratio. This makes manual correction possible while discouraging casual inflation.

Suggested default: consume `20%` of makeup study minutes as slack minutes, rounded up, with admin-configurable ratio.

If the user has insufficient slack balance, the system may either reject the makeup or mark it as unpaid/pending according to configuration. MVP should reject insufficient balance for makeup that requires cost.

## Time Limits

Single corrected sessions should follow scheduling constraints from `task-scheduling-improvements`, including max 8 hours. Daily effective study minutes should have a configurable warning threshold for abnormal records.

## Suspicious Records

Suspicious study records should be marked for admin visibility. They should not automatically ban users. Admins can review abnormal makeup, excessive minutes, or repeated suspicious behavior.

## Slack Value

Slack minutes represent rest/activity balance only. They are not cash, points with monetary value, or redeemable benefits.

## Rest Balance Suggestions

The app should use slack/study history to show simple suggestions such as `本周学习较多，可以安排 30 分钟休息`. This is guidance, not a mandatory rule.
