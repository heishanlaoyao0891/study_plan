## Why

Banned users currently receive inconsistent responses across middleware and login paths, including business errors sent with HTTP 200 and responses without enough timing metadata for a trustworthy client experience. H5 and the mini program have no dedicated safe destination, so users see generic errors or can remain on business pages while every request fails.

## What Changes

- Standardize active-ban responses across authenticated requests, H5 login/link, WeChat login, and administrator login.
- Auto-clear expired bans before authentication continues and define server-owned permanent-ban semantics.
- Preserve a safe ban payload and retained session token in the UniApp client while routing all ban responses to one dedicated page.
- Add a friendly responsive banned-account page with reason fallback, server-aligned countdown, expiry refresh, logout, and support guidance.
- Cover timed, permanent, expired, middleware, H5, WeChat mock, and link behavior with backend tests.
- Make plan delay preview-first so schedule conflicts can be corrected before any task is moved.
- Keep AI generation responsive through bounded optional enrichment and Chinese-only plan introductions.

## Capabilities

### Modified Capabilities

- `user-auth`: Standardize ban enforcement and add the H5/mini-program paused-access experience.
- `admin-config`: Define permanent-ban representation as server-owned behavior exposed explicitly to clients.
- `plan-management`: Preview delayed tasks, expose conflicts, and apply user-edited schedules atomically.
- `ai-plan-generator`: Bound optional enrichment and keep generated introductions in Simplified Chinese.

## Impact

- Authentication middleware and all credential login/link handlers.
- Ban response contract and permanent sentinel ownership.
- UniApp request handling, launch routing, local session state, and a new banned page.
- Backend authentication tests and frontend production builds.
- Plan shift preview/apply routes, schedule validation, AI enrichment budgets, and generated plan copy.
