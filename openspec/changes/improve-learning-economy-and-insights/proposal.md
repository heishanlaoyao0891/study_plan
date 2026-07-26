## Why

Several core workflows currently violate user expectations: one-hour AI plans can silently shrink to 40 minutes, fully contained schedule overlaps are too permissive, group data failures look like blank pages, slack sessions cannot create recoverable debt, and statistics do not expose useful time or plan comparisons.

## What Changes

- Preserve the first requested hour in AI-generated tasks while retaining adaptive pacing for larger workloads.
- Add a stricter 30-minute limit when one task is fully contained by another task, alongside the existing 60-minute cumulative overlap limit.
- Make group loading resilient, fix invitation routing, add empty states, and support mini-program sharing.
- Allow completed slack sessions to create signed debt, block new sessions while non-positive, repay debt through check-in rewards, and support low-balance subscription reminders.
- Add 7-day, 1-month, and 1-year statistics across time and plan dimensions with study, planned, overtime, and completion metrics.
- Upgrade the operations overview to four visual dimensions and expose user OpenID plus last-login information in user management.

## Capabilities

### Modified Capabilities

- `ai-plan-generator`
- `plan-management`
- `slacking-system`
- `notification`
- `stats-analysis`
- `admin-config`

## Impact

- AI local planning profile and duration selection.
- Backend and frontend schedule validators.
- Group invitation links and group page lifecycle.
- Slack balance ledger semantics and subscription configuration.
- Statistics API and H5/mini-program visualization.
- User login metadata and the administrator operations dashboard.
