## 1. Admin Navigation Stability

- [x] 1.1 Add a layout regression test or reproducible viewport check covering overview and the widest admin route.
- [x] 1.2 Fix the dashboard grid/sidebar inline size and constrain workspace, table, chart, loading, and error overflow so route content cannot widen navigation.
- [x] 1.3 Verify admin navigation labels, active state, overview metrics, and wide data pages at supported desktop widths.

## 2. User Utility Information Architecture

- [x] 2.1 Remove `账号与数据`, `设置与说明`, their handlers, and obsolete styles from the plan page.
- [x] 2.2 Add a separated bottom utility section to the statistics tab with `账号与数据` before `设置与说明`, reusing the existing destination pages.
- [x] 2.3 Add or update navigation tests to prove the plan tab excludes the utilities and the statistics tab routes both entries correctly.

## 3. H5 Authentication Layout

- [x] 3.1 Replace the three equal H5 mode tabs with one full-width login button, a password-label-aligned `忘记密码？` link, a centered invitation-registration entry below the button, and low-emphasis return-to-login links in registration/reset modes.
- [x] 3.2 Preserve existing H5 validation, invitation registration, administrator-assisted reset, loading, and error behavior in the recomposed forms.
- [x] 3.3 Keep the desktop auth panel approximately 400-440 CSS pixels wide with aligned field/button heights and restrained radius/shadow styling.
- [x] 3.4 Verify the H5 authentication layout, focus order, text wrapping, stable panel positioning, and action hierarchy on narrow mobile and desktop browser viewports.

## 4. Automatic Mini-Program Authentication

- [x] 4.1 Extract a testable mini-program startup authentication state machine covering loading, authenticated, setup-required, banned, exchange-error, and retry states.
- [x] 4.2 Run `uni.login` and WeChat code exchange automatically during mini-program launch, hold protected navigation until resolution, and route linked accounts directly into the application.
- [x] 4.3 Render only the first-use account setup flow for unresolved OpenIDs, with existing-account linking and invitation-backed account creation; accept launch-carried invitation context or manual invitation entry.
- [x] 4.4 Exclude H5 login/register/reset controls from the mini-program bundle and remove the visible `微信登录` landing action from normal startup.
- [x] 4.5 Add backend and frontend tests for linked login, unresolved setup token, link conflict, invitation redemption, banned account, exchange failure, retry, and setup-token rejection by study APIs.

## 5. Durable AI Generation Jobs

- [x] 5.1 Add the AI plan generation job model, migration, status constraints, owner/idempotency indexes, active-job uniqueness protection, lease fields, result plan reference, safe error fields, and generation metadata.
- [x] 5.2 Extract the existing planning Agent pipeline into a worker-callable service that builds local output, performs bounded enrichment, applies fallback, and revalidates current schedule and workload constraints.
- [x] 5.3 Implement atomic pending/expired lease claims, lease renewal, bounded attempts, startup recovery, periodic scanning, and clean worker shutdown without duplicate processing.
- [x] 5.4 Persist plan, tasks, generation source, job success, and result plan ID atomically; mark terminal failures without leaving partial plan data.
- [x] 5.5 Add authenticated `POST /api/ai/plan-jobs`, current-job lookup, and job-by-ID status endpoints with normalized input, prompt length limits, ownership checks, `202` responses, active-job reuse, and submission idempotency.
- [x] 5.6 Pass optional `追加说明` through the bounded planning prompt while proving it cannot override schedule, ownership, workload, quota, or structural validation.
- [x] 5.7 Add backend tests for submit latency, state transitions, owner isolation, duplicate submission, concurrent claims, expired lease recovery, provider fallback, retry exhaustion, process restart recovery, and atomic no-partial-plan behavior.

## 6. AI Client Migration

- [x] 6.1 Replace preview, provenance, regeneration, and commit API types with generation job submission/status types and server-authoritative active-job restoration.
- [x] 6.2 Rebuild the AI plan page as a submission form with goal, availability, and optional detailed `追加说明`, plus pending, running, failed, and succeeded states without preview or confirm-save controls.
- [x] 6.3 Show active generation status on the plan-page AI entry, prevent duplicate submission, poll only while relevant pages are visible, and stop polling on terminal state.
- [x] 6.4 Refresh the plan list and route users to the normally editable generated plan after success while preserving completion if the user navigates away or restarts the client.
- [x] 6.5 Remove client preview storage and obsolete preview editor code, then retire legacy regenerate/commit backend contracts after compatibility usage is confirmed absent.

## 7. Verification And Reconciliation

- [x] 7.1 Reconcile overlapping unfinished tasks and delta specs in `add-unified-h5-and-wechat-account-auth` and `build-constrained-planning-agent` before any affected change is archived.
- [x] 7.2 Run backend unit/integration tests, admin type-check/build, frontend type-check, H5 build, and WeChat mini-program build.
- [x] 7.3 Perform H5 desktop/mobile visual checks, mini-program launch/setup checks, admin overview/sidebar checks, and an end-to-end asynchronous generation run including navigation away and backend restart.
- [x] 7.4 Confirm no plaintext invitation, OpenID, provider secret, detailed internal error, or cross-user job data is exposed in client storage, logs, or API responses.
