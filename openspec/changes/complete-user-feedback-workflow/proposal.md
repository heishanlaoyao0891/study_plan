## Why

The product already stores feedback and exposes an administrator inbox, but the user form is hidden inside the settings-and-policy page and the workflow stops after submission. H5 and mini-program users need a discoverable support entry, clear submission feedback, and a way to see whether their reports were received or handled. Administrators need enough workflow controls to triage and respond instead of only reading a static table.

## What Changes

- Add a dedicated feedback and problem-report page shared by H5 and the mini program.
- Expose feedback from account/settings and retain a lightweight entry from the existing settings-and-policy page.
- Replace free-form categories with clear issue, suggestion, content, account, and other choices.
- Let authenticated users list their own reports and see status, administrator response, and update time.
- Add bounded input validation, duplicate-submit protection, loading, success, empty, and failure states.
- Let administrators filter reports, inspect user context, add a response, and move reports through open, processing, resolved, and closed states.
- Keep image attachments, live customer-service chat, and outbound response notifications out of scope.

## Capabilities

### Modified Capabilities

- `operations-compliance`: Complete the user feedback submission and administrator handling workflow across H5, mini program, backend, and admin console.

## Confirmed Decisions

- Feedback requires an authenticated account so reports can be tracked safely and shown back only to their owner.
- H5 and mini program use the same page, API contract, categories, and status language.
- Contact information remains optional because the in-product report history carries administrator responses.
- Administrator responses are visible to the report owner but cannot expose internal notes or other users' data.
- The initial workflow is asynchronous and text-only; it is not a real-time support channel.
