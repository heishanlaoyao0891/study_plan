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
AI_PROVIDER=mock
AI_API_KEY=<optional-ai-key>
AI_BASE_URL=<optional-ai-base-url>
AI_KEY_ENCRYPTION_SECRET=<optional-server-side-secret>
```

3. Run the backend behind a process manager such as `systemd`, listening on `127.0.0.1:8080`.

4. Put Nginx or another reverse proxy in front of it. `deploy/nginx.study-plan.conf` contains a minimal HTTP proxy example. Production mini programs require HTTPS, so terminate TLS before exposing the API domain to WeChat.

## Mini Program

1. Build WeChat output:

```bash
cd frontend
npm run build:mp-weixin
```

2. Open WeChat DevTools and import `frontend/dist/build/mp-weixin`.

3. In WeChat Mini Program admin, add the HTTPS backend domain to request legal domains.

4. Set `VITE_API_BASE=https://your-api-domain` before building, or set it from the login debug field during local testing.

## Admin Console

1. Build the PC admin console as static assets:

```bash
cd admin
npm run build
```

2. Set `VITE_ADMIN_API_BASE=https://your-api-domain` when the admin console is served from a separate domain.

3. Serve `admin/dist` behind HTTPS on a separate admin domain such as `admin.example.com`. Do not link this console from the mini program.

## Notes

- Personal mini programs can publish this app without WeChat Pay because the product uses slack-time points, not real-money payments.
- Real AI integration is currently represented by provider configuration and mock generation. Replace the mock provider before production AI billing is needed.
- SQLite is suitable for personal/friends usage. Move to MySQL/PostgreSQL before wider public usage.
