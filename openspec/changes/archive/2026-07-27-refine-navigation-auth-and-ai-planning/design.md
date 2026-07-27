## Context

The admin console uses a two-column CSS grid, but its navigation track is not protected against intrinsic content sizing. The UniApp login page currently combines H5 and mini-program variants in one component: H5 uses three equal tabs, while the mini program requires a visible tap before exchanging its WeChat code. The plan tab also contains account and help utilities unrelated to plan work.

AI planning currently runs the complete local and provider pipeline inside one HTTP request, returns an editable signed preview, and waits for a second commit request. This conflicts with the desired background experience and makes navigation or request interruption part of the generation lifecycle. The backend already owns deterministic planning, validation, quota, and transactional commit behavior, so the durable job should orchestrate those existing services rather than move generation logic into the client.

This change overlaps the active `add-unified-h5-and-wechat-account-auth` and `build-constrained-planning-agent` changes. Their shared-account identity model and constrained planner remain foundations; this change revises the client entry flow and replaces the preview lifecycle after those foundations are present.

## Goals / Non-Goals

**Goals:**

- Keep admin navigation compact and invariant across routes.
- Give H5 login, registration, and recovery conventional visual hierarchy.
- Authenticate mini-program users in the launch path without a dedicated login landing screen.
- Show account setup only for an unresolved OpenID, while supporting both existing-account linking and invitation-backed creation.
- Keep plan pages focused on plans and group personal utilities at the bottom of the statistics tab.
- Let AI generation survive page navigation, client disconnects, and backend restart, with one observable state and direct plan creation.
- Preserve validation, quota, scheduling, fallback, and normal plan editing behavior.

**Non-Goals:**

- Public registration, passwordless H5 authentication, or self-service recovery channels.
- Automatic merging of two existing application accounts.
- A general-purpose distributed queue or new message broker.
- Editing tasks while generation is still running or restoring the removed AI preview editor.
- Redesigning the statistics charts, plan cards, or all admin visual styles.

## Decisions

### Constrain admin navigation independently of workspace content

Use an explicit fixed sidebar track and fixed inline size, with `min-width: 0` and overflow constraints on the workspace and its data regions. Navigation labels remain single-line or ellipsized inside their own track. This addresses intrinsic grid sizing rather than applying route-specific styles to the overview.

Route-specific widths were rejected because another wide table or chart could reproduce the same defect.

### Give H5 authentication one primary path

H5 opens on login with one full-width primary submit button below the username and password fields. `忘记密码？` is a small text link aligned to the right of the password label. `还没有账号？使用邀请码注册` is centered below the primary button, with only the registration phrase using the brand accent. The three modes are never rendered as equal segmented tabs.

Selecting registration replaces the form in the same stable-width panel with invitation, username, nickname, and password fields, one full-width `注册并开始学习` primary button, and a low-emphasis `已有账号？返回登录` link below it. Selecting password reset replaces the form with username, administrator reset code, and new password fields, one full-width `重置密码` primary button, and a low-emphasis back-to-login link. The panel remains approximately 400-440 CSS pixels wide on desktop, uses aligned input and button heights, permits natural wrapping on mobile, and avoids exaggerated pill shapes or heavy shadows.

Separate pages were considered, but retaining one route and form component preserves current UniApp routing and validation with a smaller migration.

### Run WeChat authentication during mini-program launch

The mini-program launch path calls `uni.login`, exchanges the code, and routes based on the server response before showing authenticated tabs. A linked active OpenID receives the normal session and continues without a login choice. An unresolved OpenID receives only a short-lived setup token and is routed to a first-use account setup view. The setup view can link an existing account using username/password or create an account using username/password/nickname and an invitation from the launch payload or manual input.

The app must distinguish startup loading, authenticated, setup-required, banned, and recoverable-error states so protected pages are not briefly exposed. A retry state is shown for network or WeChat exchange failure; H5 login/register/reset controls are excluded from the mini-program build.

Silently creating an account for every OpenID was rejected because invitation control and cross-platform account ownership would be bypassed.

### Put personal utilities at the end of the statistics tab

Remove account and help entries from the plan template and handlers. Add a visually separated utility section after all statistics content, ordered `账号与数据` then `设置与说明`. This keeps the rightmost tab as the conventional personal/summary destination while preserving existing destination pages.

Adding another tab was rejected because the current rightmost statistics tab can own these low-frequency utilities without changing tab count.

### Persist AI generation as a leased job

Introduce an `ai_plan_generation_jobs` record containing user ID, normalized request JSON, status (`pending`, `running`, `succeeded`, `failed`), attempt count, lease timestamps, resulting plan ID, bounded public error code/message, source metadata, and timestamps. Only one active job per user is accepted; duplicate submission returns the existing active job. A client idempotency key additionally makes retries stable.

`POST /api/ai/plan-jobs` validates and stores input, then returns `202 Accepted` with the job representation. `GET /api/ai/plan-jobs/current` and `GET /api/ai/plan-jobs/:id` return only jobs owned by the authenticated user. The frontend restores state from the server, not local memory, and polls with bounded intervals while visible.

An in-memory goroutine per request was rejected because jobs would be lost during restart and could not be reliably observed from another client session. A broker was rejected as unnecessary operational complexity for the current single-backend deployment.

### Reuse the constrained planner and commit server-side

The worker claims a pending or expired-running job with a lease, builds planning context, produces and validates the local candidate, optionally enriches it, and persists the final plan and tasks. Final overload and schedule checks run against current data immediately before persistence. Job success and resulting plan ID are committed atomically with plan creation, so a succeeded job always references a visible plan.

The optional `追加说明` is passed as untrusted user preference text to the existing delimited planning prompt. It can shape content within backend-owned constraints but cannot override availability, conflicts, quotas, ownership, or validation.

Preview provenance and client commit are removed from the primary flow because the candidate never crosses a client trust boundary before persistence. Users make later corrections through the existing plan detail/edit API.

### Make failures retryable without duplicating plans

Workers renew leases during processing. Startup and periodic scans reclaim expired `running` jobs up to a bounded attempt limit. Provider failure still permits the validated local plan according to existing fallback policy. Terminal validation or persistence failure marks the job `failed` with a safe user-facing message; resubmission creates a new job after the prior job is terminal. Idempotency and transactional result-plan binding prevent duplicate plans.

## Risks / Trade-offs

- [Launch authentication adds startup latency] -> show a neutral startup state, keep code exchange bounded, and expose retry only on actual failure.
- [Launch-carried invitations can leak] -> keep invitations single-use and expiring, validate only on the server, and do not persist plaintext after redemption.
- [An in-process worker can pause while all instances are down] -> durable pending jobs resume on startup; no job is represented as complete before persistence.
- [Multiple backend instances can process the same job] -> use an atomic lease claim, idempotency key, and transactional result binding.
- [Direct persistence removes pre-save review] -> enforce deterministic constraints before save and route users to normal plan editing after completion.
- [Polling adds requests] -> expose one compact current-job endpoint, poll only for active jobs while relevant pages are visible, and stop at terminal state.
- [Active changes still describe previews or visible WeChat login] -> reconcile their remaining tasks and specs before archive so the final main specs contain only the revised behavior.

## Migration Plan

1. Stabilize admin sizing and relocate frontend utility entries independently.
2. Add AI job schema, indexes, ownership checks, lease claim logic, worker lifecycle, and API tests without removing old routes.
3. Add job submission/status APIs and deploy backend worker support.
4. Migrate the UniApp AI flow to jobs, direct completion, and plan refresh; stop invoking preview commit routes.
5. Change mini-program launch routing to automatic code exchange and conditional setup while keeping H5 authentication route behavior platform-specific.
6. After deployed clients no longer call them, remove preview/regenerate/commit UI and retire legacy commit contracts and provenance storage.
7. Rollback can restore the prior client while compatibility routes remain; queued jobs may be paused safely because their state is durable.

## Open Questions

- None. Initial implementation uses the existing backend process with a durable database lease worker; a broker can be introduced later only if deployment scale requires it.
