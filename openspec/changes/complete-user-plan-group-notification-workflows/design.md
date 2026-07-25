## Context

The application uses one account across H5 and WeChat, concrete `DailyTask` rows for execution, and a Go API with SQLite. Existing APIs cover much of plan and group mutation but the client surfaces and lifecycle invariants are incomplete. Notification delivery currently has configuration and a sender but lacks platform authorization, scheduling, and idempotency.

## Decisions

### Account state is authoritative on the server

Users store onboarding status/version/completion time and a security version. Access JWTs carry the security version and middleware rejects stale versions or inactive/deleted accounts. Local logout removes only the local token. Password change verifies the current password, increments the security version, and returns a fresh token. Admin reset codes are random hashed records with one use and a 30-minute expiry; redemption sets a new bcrypt password and increments security version.

### Manual planning edits concrete task drafts

The create request accepts explicit task drafts with date, title, objective, description, start, and end. Every task date must fit the plan range and selected study dates. The backend validates the complete proposed schedule and creates plan/tasks atomically. A plan detail page owns later plan and task edits. Completed or active tasks retain stricter mutation constraints.

### Check-in remains truthful to today

Check-in APIs continue returning only the requested date. When no rows exist, the response adds the nearest future pending task and plan summary. The client explains the next scheduled date and links to plan detail.

### Groups enforce lifecycle semantics

Any active member can create invitations. Leaders can remove members; destructive actions require confirmation. Expired end dates transition active groups to ended before reads/mutations. Weekly leaderboard values query the current Shanghai week. Membership transitions execute transactionally with database-supported guards where practical.

### WeChat authorization precedes delivery

The backend publishes enabled reminder types and template IDs. The mini program calls `requestSubscribeMessage` and submits each accepted/rejected result. H5 renders an unsupported-channel explanation. A background worker periodically calculates due events, claims a unique delivery key, sends schema-compatible payloads, and records provider outcomes. Group nudges use the same delivery pipeline. Access tokens are cached until shortly before expiry.

## Risks / Trade-offs

- Explicit task drafts increase create-form complexity -> generate sensible rows from selected dates and provide bulk-fill controls.
- Token versioning logs users out after security changes -> return a fresh token for the current password-change session and explain other-device logout.
- WeChat templates differ in field names -> store per-reminder JSON field mapping/page configuration and validate before enabling.
- Scheduler duplication across replicas -> enforce unique event keys in the database before sending.
- Group lifecycle races -> transactions plus uniqueness checks; return domain conflicts on races.

## Migration Plan

1. Add nullable/defaulted user lifecycle columns, reset-code records, and notification delivery keys.
2. Deploy backward-compatible response fields and endpoints.
3. Deploy client onboarding/account and plan-detail flows.
4. Enable notification worker only after approved template IDs and mappings are saved.
5. Remove obsolete plan action-layout UI while retaining stored rows harmlessly.
