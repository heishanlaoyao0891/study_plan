## Context

`POST /api/feedback` already creates a `FeedbackReport`, the operations page contains an inline form, and the admin console lists the latest one hundred reports. Reports currently contain category, content, optional contact, and an unused status field. There is no owner list endpoint, response field, status mutation endpoint, filtering, or prominent account/settings entry.

## Decisions

### Feedback becomes a dedicated shared page

The frontend adds one page used by both H5 and the mini program. Account/settings provides the primary entry and the operations page links to it instead of embedding a second form. The page presents fixed category choices, a bounded text area, optional contact, submit progress, and a report history below the form.

### Reports have a small explicit lifecycle

Reports use `open`, `processing`, `resolved`, and `closed`. New reports start open. Administrators may update status and deliberately set or clear a user-visible response. A status-only update preserves the existing response and response metadata; setting a response refreshes responder metadata, while explicit clearing removes it. The owner list endpoint returns only the authenticated user's reports and public response fields.

### Server validation remains authoritative

The backend allowlists categories, trims input, enforces content/contact lengths, and rejects empty content. The client mirrors these rules for immediate feedback and disables repeated submission while a request is active. A SQLite trigger atomically enforces the short authenticated rate limit at insertion time, including concurrent submissions.

### Admin triage stays lightweight

The admin inbox supports status/category filters, report detail, user identifier, optional contact, response editing, and status actions. Internal notes and assignment queues are deferred until there is a concrete operations need.

## Risks / Trade-offs

- A prominent entry may increase low-quality reports -> use categories, concise guidance, validation, and rate limiting.
- User-visible responses create copy and privacy risk -> expose only a dedicated public response and keep actor metadata server-side.
- Existing free-form categories may not match the new allowlist -> preserve legacy rows for display while applying the allowlist only to new submissions.
- Existing reports have no response metadata -> use nullable migration-safe fields and retain their current open status.

## Migration Plan

1. Add nullable public response, responded-at, and responding-admin fields to feedback reports.
2. Deploy owner-list and admin mutation endpoints with backward-compatible existing list/submit behavior.
3. Deploy the dedicated H5/mini-program page and replace the inline operations form with navigation.
4. Deploy admin filtering, detail, response, and lifecycle controls.
