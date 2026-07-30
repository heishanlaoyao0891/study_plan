## Context

Study sessions are already persisted with a server timestamp and a database constraint allowing one open session per user. The timer task views calculate current elapsed time server-side. Authentication currently always routes to onboarding or the check-in tab, so a newly authenticated H5 session does not discover an open task, especially when it began on another client or before midnight.

## Goals / Non-Goals

**Goals:**

- Provide an authenticated, user-scoped way to discover the sole active study task.
- Route an authenticated user to that task before the normal home route.
- Preserve the original session and its elapsed duration without a restart request.
- Use one recovery helper for H5 and WeChat authentication flows.

**Non-Goals:**

- Allow simultaneous study sessions or multi-device control conflicts.
- Change login invalidation policy, session duration, or automatic midnight closure.
- Restore paused, completed, or historical tasks automatically.

## Decisions

- **Add `GET /api/tasks/active` rather than infer from the daily check-in list.** An active session can originate on another date before midnight compensation and does not have to be on today's check-in list. The endpoint returns `null` when none exists and a timer view with task data when one exists.
- **Use the existing timer view builder.** This keeps `accumulated_seconds`, target duration, state, and session timestamp consistent with task detail and check-in responses, instead of duplicating elapsed-time calculations in the client.
- **Resolve navigation after a token is stored.** The shared client helper first respects onboarding requirements, then fetches the active task, and falls back to `/pages/checkin/checkin` if none exists or recovery lookup has a transient failure. A lookup failure must not block a successful login.
- **Navigate directly to task detail without calling start/resume.** The task page reloads server state and ticks only from the loaded snapshot, so recovery cannot create a second session or reset elapsed time.

## Risks / Trade-offs

- [A failed recovery lookup could make login feel broken] → Fall back to the established check-in route and leave the existing task visible there.
- [An open session spanning midnight is stale] → Existing midnight compensation remains authoritative and runs before normal navigation; the active endpoint returns only an actually open row.
- [A client holds an invalid token] → Existing 401 handling clears credentials; recovery makes no special exception.

## Migration Plan

1. Deploy backend route and both-client build together.
2. Start a task on one client, authenticate on the other, and confirm the task view shows the original accumulated duration.
3. Roll back client navigation independently if needed; the new endpoint is additive and does not alter stored sessions.

## Open Questions

- None.
