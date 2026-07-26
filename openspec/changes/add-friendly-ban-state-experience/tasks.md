# Tasks

## 1. Backend Contract

- [x] Centralize the safe real-403 ban response envelope.
- [x] Move permanent sentinel creation and recognition into the model.
- [x] Auto-clear expired bans before authentication proceeds.
- [x] Apply one check to middleware, H5, WeChat, link, and admin login paths.
- [x] Prevent banned WeChat linking from mutating account identity.

## 2. Client State and Routing

- [x] Validate and persist only safe ban response metadata.
- [x] Detect HTTP and business-envelope ban responses globally.
- [x] Preserve tokens and guard against banned-page re-launch loops.
- [x] Route startup and tokenless login bans through shared interception.
- [x] Recover retained sessions through `/auth/me` and `routeForUser`.

## 3. Banned Experience

- [x] Register a responsive H5 and mini-program banned page.
- [x] Show reason fallback, absolute time, and live timed countdown.
- [x] Show permanent wording without a countdown.
- [x] Reconcile server time across foreground and background lifecycle.
- [x] Handle malformed state, timer cleanup, logout, and tokenless retry.
- [x] Provide safe support guidance without an unusable feedback link.

## 4. Tests and Verification

- [x] Add middleware timed-ban and expired auto-clear tests.
- [x] Add H5 timed/permanent metadata and expiry tests.
- [x] Add WeChat mock and banned-link mutation tests.
- [x] Run gofmt, Go tests, and Go build.
- [x] Run frontend type-check, H5 build, and WeChat build.
- [x] Run strict OpenSpec validation and final security/loop diff review.

## 5. Experience Hardening

- [x] Bound persisted server timestamps before accepting ban state.
- [x] Clear any residual credential when a tokenless ban returns to login.
- [x] Refine the banned page into a compact responsive status view.

## 6. Adjacent Workflow Fixes

- [x] Add editable plan-delay preview and authoritative apply validation.
- [x] Avoid transient unique-date failures while moving consecutive tasks.
- [x] Bound optional AI enrichment so a complete local plan remains available.
- [x] Keep generated plan introductions and capacity warnings in Chinese.
