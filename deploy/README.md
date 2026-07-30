# Study Plan Deployment

## Backend

1. Build on the server or locally for the server platform:

```bash
cd backend
go build -o study_plan_backend .
```

2. Create `.env` from `backend/.env.production.example` and set production values:

```bash
APP_ENV=production
PORT=8080
DB_PATH=/var/lib/study-plan/study_plan.db
JWT_SECRET=<strong-random-secret>
ADMIN_USERNAME=<admin-username>
ADMIN_PASSWORD=<initial-admin-password>
WECHAT_APPID=<mini-program-appid>
WECHAT_SECRET=<mini-program-secret>
WECHAT_LOGIN_MOCK=false
AI_KEY_ENCRYPTION_SECRET=<strong-random-server-side-secret>
AVATAR_STORAGE=minio
AVATAR_BASE_URL=https://assets.slls.asia/study-plan-assets
ARCHIVE_ENABLED=false
ARCHIVE_DRIVER=mysql
ARCHIVE_DSN=<mysql-user>:<mysql-password>@tcp(<mysql-host>:3306)/study_plan_archive?charset=utf8mb4&parseTime=True&loc=Local
ARCHIVE_INTERVAL_MINUTES=5
ARCHIVE_TABLES=users,plans,daily_tasks,checkins,study_sessions,slack_records
```

Before release, verify the WeChat phone-number component capability, account qualification, cost, and quota in the WeChat Mini Program console.

3. Run the backend behind a process manager such as `systemd`, listening on `127.0.0.1:8080`.

4. Put Nginx in front of loopback-bound services. `deploy/nginx.slls.conf` is the canonical Docker deployment and `deploy/nginx.study-plan.conf` is the direct-host alternative. Both redirect HTTP to `https://slls.asia`; do not expose a raw-IP test port.

5. In the admin AI configuration, select the recommended `siliconflow` provider, keep the pinned Base URL `https://api.siliconflow.cn/v1`, use model `deepseek-ai/DeepSeek-V3.2`, and enter a SiliconFlow API key. The backend sends OpenAI-compatible `POST /chat/completions` requests and requires `AI_KEY_ENCRYPTION_SECRET` to store the key securely.

### SQLite backup

Use the backup script before risky changes and at least once per day in production.

```bash
cd backend
powershell -File scripts/backup-sqlite.ps1 -Source study_plan.db -BackupDir backups
```

Restore by stopping the backend, replacing the database file with the latest backup copy, then starting the backend again.

### Archive sync

When `ARCHIVE_ENABLED=true`, the backend copies selected SQLite tables to a MySQL archive database every `ARCHIVE_INTERVAL_MINUTES` minutes. SQLite remains the source of truth.

Archive failures are written to backend logs and do not block normal study, check-in, or slack operations.

### Tencent Cloud release checks

- Confirm `slls.asia` and `www.slls.asia` resolve to the production server and the certificate chain is valid.
- Verify `https://slls.asia/`, `https://slls.asia/admin/`, and `https://slls.asia/health`.
- Add `https://slls.asia` to WeChat request legal domains.
- Confirm `WECHAT_LOGIN_MOCK=false` in production.
- Confirm the WeChat phone-number component account qualification, cost, and quota before release.
- Schedule a daily SQLite backup and run a manual backup before deployments.
- If using same-server MinIO in Docker for avatars, mount persistent volumes and include those volumes in the server backup plan.

## Mini Program

1. Build and verify the production WeChat release candidate:

```bash
cd frontend
npm run release:mp-weixin
```

2. Open WeChat DevTools and import `frontend/dist/build/mp-weixin`.

3. In WeChat Mini Program admin, add `https://slls.asia` to request legal domains.

4. Complete `docs/wechat-submission-checklist.md`, then upload version `1.0.1` from WeChat DevTools. Production builds read `frontend/.env.production` and reject IP, HTTP, or development-overridable API settings.

## Admin Console

1. Build the PC admin console as static assets:

```bash
cd admin
npm run build
```

2. Keep `VITE_ADMIN_API_BASE` empty for the current same-origin deployment.

3. Serve `admin/dist` at `https://slls.asia/admin/`. Do not link this console from the mini program.

## Rollback

Before deploying, record the current Git revision and run the SQLite/object-storage backup. If smoke checks fail, redeploy the previous revision with the existing `.env.production` and named Docker volumes. Do not delete or recreate `study-plan-data` or `study-plan-minio-data` during an application rollback.

Use loopback diagnostics over SSH when needed:

```bash
curl --fail http://127.0.0.1:8080/health
curl --fail --resolve slls.asia:443:127.0.0.1 https://slls.asia/health
```

## Notes

- Personal mini programs can publish this app without WeChat Pay because the product uses slack-time points, not real-money payments.
- AI planning supports the recommended SiliconFlow preset, generic OpenAI-compatible providers, and deterministic mock fallback. Run the structured provider test before enabling production traffic.
- SQLite is suitable for personal/friends usage. Move to MySQL/PostgreSQL before wider public usage.
- If MinIO is used for avatars, keep it behind HTTPS and private to the backend. The mini program should only receive public or signed URLs.
