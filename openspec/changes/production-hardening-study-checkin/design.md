# Design: Production Hardening

## Runtime Modes

The backend SHALL distinguish local development from production using explicit environment variables. Production startup should fail fast when required secrets are missing.

Required production values:

- `JWT_SECRET`
- `WECHAT_APPID`
- `WECHAT_SECRET`
- `WECHAT_LOGIN_MOCK=false`
- `DB_PATH`

Deployment target is Tencent Cloud. Local development and validation remain required before deployment. The same binary should support local mock mode and Tencent Cloud production mode through environment configuration.

## WeChat Login

Local mock login remains available only when `WECHAT_LOGIN_MOCK=true`. Production login exchanges `code` for `openid` through WeChat `jscode2session` and never accepts mock openids.

The mini program should perform silent login through `uni.login()` when possible to obtain backend identity. Phone number collection cannot be silent in WeChat; it requires an explicit user tap on a phone authorization button. Because WeChat phone-number verification is not available to personal-subject mini programs, the app SHALL keep phone binding optional by default. Certified non-personal deployments MAY set `PHONE_BINDING_REQUIRED=true` to block study features until the user binds a verified phone number.

Avatar collection should be opportunistic. If the mini program can obtain or let the user choose an avatar during a low-friction profile flow, the app stores it. If avatar collection is skipped or unavailable, it SHALL NOT block login or study features.

Phone number binding may be used later for PC account matching or login, but current PC user login is not in scope.

Avatar binary data SHALL NOT be stored directly in the relational database. The database stores only an avatar URL or object-storage key. For local development this can be a URL returned by the mini program or a local/static path. For Tencent Cloud production, uploaded avatars should be stored in an object-storage compatible service such as Tencent Cloud COS or a self-hosted MinIO service running in Docker.

If using self-hosted MinIO, the deployment must expose avatar files through HTTPS and should keep MinIO credentials private to the backend. The mini program should receive only public or signed avatar URLs, never MinIO access keys.

For initial Tencent Cloud deployment, self-hosted MinIO may run on the same server as the backend through Docker, with persistent volumes and backup included in the server backup plan.

Avatar binding is expected to have no per-use platform fee. Phone number authorization depends on WeChat's current capability rules and account qualification. Public WeChat documentation says personal-subject mini programs cannot rely on the phone-number verification capability; certified non-personal deployments should verify fees, quota limits, and eligibility in the WeChat console before enabling `PHONE_BINDING_REQUIRED=true`.

## Notification Delivery

The current notification due-event endpoint becomes the source for sending subscription messages. A sender layer maps event types to template IDs and records delivery attempts.

WeChat subscription messages are not ordinary push notifications. A user must actively authorize a specific message template before the app can send that type of message. The backend should only attempt delivery when the user has granted subscription permission and the template is configured/enabled.

## Operations

Structured logs should include request id, user id when available, route, status, latency, and error message. Sensitive values such as JWTs, AppSecret, and API keys must not be logged.

## Backup

SQLite remains acceptable for personal/friends usage, but production deployment must include a backup command or documented job that copies the database safely.

The system should also support an optional archival sync to MySQL. SQLite remains the source of truth. When archive sync is enabled, selected tables are copied to MySQL for backup/reporting/migration readiness. Failure to archive should be logged and visible, but should not block normal user study/check-in operations.

The archive sync should run on a configurable interval and default to 5 minutes when enabled. It should be idempotent where possible.

SQLite backups should run at least daily in production, with a manual backup command available before risky operations or deployments.

Suggested archive configuration:

- `ARCHIVE_ENABLED=false`
- `ARCHIVE_DRIVER=mysql`
- `ARCHIVE_DSN=`
- `ARCHIVE_INTERVAL_MINUTES=5`
- `ARCHIVE_TABLES=users,plans,daily_tasks,checkins,study_sessions,slack_records`
