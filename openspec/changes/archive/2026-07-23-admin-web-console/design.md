# Design: Admin Web Console

## Product Boundary

The mini program SHALL contain only normal user workflows. Admin pages, admin navigation, and user management actions SHALL be removed from the mini program.

The PC admin console SHALL be a separate Vue frontend surface. It can live in the same repository, for example under `admin/`, and use the existing backend `/api/admin/...` endpoints.

Admin capabilities currently exposed in the mini program SHALL be migrated to the PC admin console. The mini program should only use normal user APIs and should not include admin pages, admin navigation, role-management UI, user-management UI, or configuration-management UI.

Recommended stack:

- Vue 3
- TypeScript
- Vite
- Vue Router
- Pinia or a small local store if state remains simple

## Authentication

The admin console SHALL use username/password login for the MVP. Passwords must be stored as secure hashes, not plaintext. Successful login returns a JWT or session token containing `role=admin`.

The initial admin account should be bootstrapped from environment variables during deployment, for example `ADMIN_USERNAME` and `ADMIN_PASSWORD`, and forced to use secure password hashing after creation. The bootstrap secret should not remain visible in logs.

All admin API requests continue to require `role=admin`. WeChat QR-code login and SMS/phone-code login are out of scope for this iteration.

## UI Scope

The initial admin console should be simple and utilitarian:

- Login page.
- Overview dashboard with key counts.
- User list with search/filter.
- User detail summary.
- Ban/unban actions with reason and duration.
- Role visibility for users, without general role editing in MVP.
- Global and per-user slack configuration.
- AI model configuration, including provider, model name, base URL, enabled state, and masked API key updates.
- WeChat subscription message configuration, including template IDs, enabled reminder types, and send status visibility.
- Audit log list for sensitive admin actions.

Later admin iterations may add user plan debugging, system health, and CSV export.

## AI Configuration

AI provider settings belong in the PC admin console. The console should allow the admin to configure provider type, model name, base URL, request timeout, daily generation limit, and API key. API keys must be write-only or masked after saving; the backend must never return the full secret value to the frontend.

AI API keys should be encrypted before persistence when a server-side encryption secret is configured. At minimum, the admin console must only display masked key status after saving.

## Subscription Message Configuration

WeChat subscription message settings belong in the PC admin console. The console should allow the admin to configure template IDs for study start reminders, completion reminders, 23:30 decision reminders, and missed check-in reminders. Each reminder type should have an enabled/disabled switch, and delivery status/failure summaries should be visible for troubleshooting.

## Audit Trail

Sensitive admin actions should be recorded with admin user id, target user id when applicable, action type, reason, and timestamp.

## Deployment

The admin console can be served as static assets behind HTTPS. It should be protected by admin login and should not be linked from the mini program.

Recommended production deployment uses a separate admin domain such as `admin.example.com`, while the mini program API remains on an API domain such as `api.example.com`.
