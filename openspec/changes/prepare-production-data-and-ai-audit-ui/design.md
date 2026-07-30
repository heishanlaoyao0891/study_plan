## Context

The backend already returns AI invocation history with `items`, `total`, `page`, and `size`, and the admin API client already models that response. The current admin page discards the pagination fields, always asks for 50 rows, and renders the table inside a panel constrained by the shared `.wide-panel` width. The table also sets a fixed 1100px minimum width, which pushes a horizontal scrollbar to the bottom of the module.

Production data cleanup is a release operation, not a new user-facing feature. It still needs the same rigor as code: identify targets with read-only queries, create a recoverable backup, delete only confirmed test users and dependent data, and record the result.

The WeChat existing-account link path already supports binding a current setup token to an H5 password account, but production can contain legacy incomplete users from earlier WeChat login behavior. Those rows have an OpenID without complete username, nickname, or password credentials. They should not permanently block the real H5 account from claiming the current WeChat identity.

Invitation-only registration is correct for the early release, but users who arrive without an invitation need an obvious, low-friction path to contact the administrator. The authentication home page is the shared unauthenticated entry for H5 and mini program setup.

## Goals / Non-Goals

**Goals:**

- Make the AI invocation history usable in the PC admin console without requiring bottom horizontal scrolling on normal desktop workspaces.
- Expose the backend pagination contract in the UI, including total count, page number, page-size selection, and previous/next controls.
- Preserve the existing refresh coupling where invocation history refresh also refreshes aggregate AI token metrics.
- Clean confirmed production test accounts only after a database backup exists.
- Allow current WeChat setup identities to bind to a valid H5 account when a legacy incomplete OpenID holder exists.
- Return a clear Chinese conflict when the current WeChat identity is already linked to another complete account.
- Show an administrator QR-code entry below the authentication card and open the QR code in a modal/preview.

**Non-Goals:**

- Add new AI invocation backend fields or change the immutable ledger model.
- Add bulk-delete user management UI.
- Delete ambiguous real-user data based only on nickname or guesswork.
- Merge full accounts or reassign user-owned learning data between complete accounts.
- Generate or infer the administrator's personal QR-code image. The actual image remains a replaceable static asset.

## Decisions

- Use the existing backend pagination API instead of adding a new endpoint. This keeps the change focused on the missing admin UI behavior and avoids changing a working audit API.
- Override only the invocation panel width with `max-width: none` and `width: 100%`. Other configuration panels should keep their current constrained form width.
- Replace the table's fixed `min-width: 1100px` with responsive column widths and wrapping. Long trace, job, model, and error content can wrap within cells because this page is an audit viewer, not an export grid.
- Keep page state local to `AIConfigView.vue`. The filter form is small, and adding a shared table abstraction would create more moving parts than this release fix needs.
- For production cleanup, use SQLite backup plus a single transaction that mirrors the existing `cleanupUserData` deletion scope before removing user rows. This makes rollback possible and prevents partial data removal.
- In `WeChatLink`, treat another row holding the setup-token OpenID as migratable only when it is incomplete: empty username or nickname credentials and no password hash. Clear that row's OpenID and mark it inactive before linking the H5 account. A complete holder remains a hard conflict.
- Add the QR entry to `login.vue` rather than a separate page. Use `/static/invite-qrcode.png` as the stable asset path so release replacement is simple and does not require code changes.

## Risks / Trade-offs

- Test-user identification ambiguity -> Mitigation: run read-only inventory first and delete only accounts with unambiguous test markers or no production legitimacy.
- Production deletion mistake -> Mitigation: create a timestamped database backup before deletion, delete in a transaction, and report exact IDs and backup path.
- Dense audit table loses some scanability when wrapping -> Mitigation: keep primary fields visible in each column and use compact secondary lines for details.
- Pagination state can go stale after filtering -> Mitigation: reset to page 1 when running a filtered query or changing page size.
- Legacy identity holder may contain useful data -> Mitigation: migrate only incomplete credential-less rows; complete holders still require manual admin handling.
- Missing QR image would make the entry visually incomplete -> Mitigation: keep the path stable and require the real personal QR image at `/static/invite-qrcode.png` before release.
