# Subscription Message Integration

The subscription implementation intentionally touches shared files that other portions of this change may also edit:

- `backend/main.go` starts `services.StartNotificationScheduler(db.DB)` after database initialization and registers `GET /api/notifications/templates`.
- `backend/models/admin_config.go` owns reminder configuration, authorization, event-key, and delivery-log persistence. Preserve these fields when merging other model work.
- `backend/models/group.go` remains the group-nudge record shape; `handlers/group.go` now stores the exact synchronous delivery outcome instead of claiming a nonexistent queue.
- `frontend/src/api/index.ts` sends WeChat's per-template authorization result map to the backend. Preserve this method signature when merging frontend API changes.
- `admin/src/api.ts` includes complete template configuration fields and now surfaces backend validation messages for non-2xx responses.

The scheduler's testable entry point is `services.RunNotificationCycle`. Both scheduler events and group nudges call `services.DeliverNotification`; the unique non-empty `notification_delivery_logs.event_key` database index is the duplicate-send guard. The WeChat sender is injected in tests, and production scheduler/group paths pass `services.SendSubscriptionMessage` directly.
