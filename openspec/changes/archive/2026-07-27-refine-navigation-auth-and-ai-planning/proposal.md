## Why

Several high-frequency entry points currently expose implementation details instead of matching user expectations: the admin sidebar can resize with overview content, H5 authentication gives three peer actions too much visual weight, mini-program users land on an explicit login screen, account utilities are mixed into plan management, and AI planning blocks on an editable preview. These issues make routine navigation and plan creation feel less predictable across PC, H5, and WeChat.

## What Changes

- Keep the PC admin sidebar at a stable, compact width when operators enter the overview or other content-heavy pages, with long content constrained inside the workspace instead of widening navigation.
- Recompose H5 authentication around one primary login form, a nearby registration action, and a lower-emphasis password-reset link rather than presenting login, registration, and reset as equal tabs.
- Make mini-program authentication automatic on launch: exchange the WeChat code in the background, enter the application immediately for a linked OpenID, and show account setup only when the OpenID cannot resolve to an eligible account.
- For an unresolved mini-program identity, support linking an existing username/password account or creating an invited account; consume a supplied or launch-carried invitation during account creation and never expose the H5 login/register/reset selector as the mini-program home screen.
- Remove `账号与数据` and `设置与说明` from the plan page and place both as bottom utility entries in the rightmost statistics tab, with account/data above settings/help.
- **BREAKING** Replace synchronous AI preview generation and explicit commit with a persisted asynchronous generation job. Submission returns promptly, generation continues on the backend, the plan is created automatically after validated output is ready, and the client observes job state and refreshes the plan list.
- Keep `追加说明` as an optional free-form prompt for the user's detailed planning preferences, but remove preview editing, regeneration, and confirmation from the generation flow; users edit the created plan through normal plan detail/edit workflows.

## Capabilities

### New Capabilities

<!-- No new standalone capability; this change refines existing product capabilities. -->

### Modified Capabilities

- `admin-config`: Require stable PC admin navigation dimensions across overview and content-heavy routes.
- `user-auth`: Separate H5 authentication action hierarchy from automatic mini-program OpenID authentication and conditional first-use account setup.
- `plan-management`: Move account/data and settings/help utilities from plan management to the bottom of the statistics tab.
- `ai-plan-generator`: Replace preview-and-commit generation with durable asynchronous jobs that directly create editable plans and expose observable generation state.

## Impact

- Admin Vue layout and shared dashboard CSS sizing and overflow behavior.
- UniApp H5 authentication layout, mini-program startup/auth routing, statistics and plan pages, and cross-platform conditional rendering.
- Authentication endpoints for WeChat code exchange, account linking, invitation-backed registration, and launch-carried invitation context.
- AI generation API contracts, job persistence/status transitions, worker execution, idempotency and quota handling, plan creation transactions, and frontend polling/resume behavior.
- Existing preview/commit clients and tests must migrate to job submission/status contracts; normal plan editing remains the post-generation correction path.
