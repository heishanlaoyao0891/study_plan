## Context

The affected workflows share one problem: client-visible behavior is derived from implicit backend rules without enough state or correction paths. The implementation should keep server validation authoritative while giving H5 and mini-program users clear previews, blocked reasons, empty states, and comparable metrics.

## Decisions

### AI duration floor

`hours_per_day` remains an upper workload input, but the first available hour is not reduced by learning-profile adaptation. Profile pacing may still reduce requests larger than one hour. Future scheduled tasks are excluded from historical completion-rate calculation.

### Containment-aware overlap

The existing union of covered minutes remains authoritative. Any task must have less than 60 covered minutes in general. When another single task fully contains its interval, the contained task may be at most 30 minutes long. Backend and frontend validators use the same rule.

### Group partial loading

Current group identity is loaded first. Members, leaderboard, and history then load independently so one failure cannot blank unrelated sections. Invitation paths target the registered group page and mini-program sharing uses that path.

### Signed slack debt

Only an already-started slack session may take the balance below zero. Settlement records the exact negative delta once in the same transaction as the balance mutation. A non-positive balance blocks new sessions; normal daily check-in rewards naturally repay debt. Low-balance reminders use the existing one-shot WeChat subscription mechanism and require an enabled administrator template plus user authorization.

### Server-owned statistics ranges

The backend defines Shanghai-calendar ranges and returns zero-filled time buckets or plan buckets. The frontend uses native view-based bars for cross-platform rendering and does not add a chart dependency.

### Operations dashboard

The administrator overview uses existing authoritative data and avoids inventing login-frequency history. Its four charts are user health segments, 30-day registrations, 30-day learning/check-in activity, and plan status distribution. User segments are mutually exclusive in this precedence: deleted, inactive, actively banned, logged in within 7 days, logged in within 8-30 days, and no login for more than 30 days. Registration time is the baseline when a user has never logged in.

Successful user token issuance records the latest login time and method. User management displays this metadata and OpenID because both are operational identity fields available only to administrators.

## Risks / Trade-offs

- Existing AI plans are not rewritten.
- Signed balances require every consumer to display negative values rather than clamp them.
- Reminder delivery remains dependent on explicit WeChat authorization.
- Task-level statistics follow persisted task dates and exclude an open session until it is paused or stopped.
