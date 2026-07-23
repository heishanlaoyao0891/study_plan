# Production Hardening for Study Check-In

## Why

The current app demonstrates the core product flow, but it still relies on local/mock behavior and lacks production safeguards. Before publishing to WeChat, the backend and mini program need real WeChat login, deployable runtime configuration, operational visibility, backup posture, and production-ready notification delivery.

## What Changes

- Replace mock login with real WeChat `code2session` behavior in production.
- Support optional phone number binding after WeChat login; keep it disabled for personal-subject mini programs and allow certified non-personal deployments to require it through configuration. Collect avatar opportunistically without blocking login.
- Add production-safe runtime configuration and startup validation.
- Add WeChat subscription message template delivery behind the existing notification event queue.
- Add structured logs for key user actions and backend errors.
- Keep SQLite as the primary lightweight database and add configurable archival/sync support to MySQL.
- Add SQLite backup guidance and basic backup execution support for small-scale deployment.
- Document deploy checks for HTTPS domains, legal request domains, and release readiness.

## Non-Goals

- No WeChat Pay integration.
- No migration away from SQLite in this change.
- No requirement to make MySQL the primary online database in this change.
- No public multi-tenant SaaS operations model.

## Impact

This change touches authentication, notifications, deployment configuration, and operational practices. It should not change core study/check-in behavior except to make production behavior explicit and safer.
