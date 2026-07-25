# CI/CD Deployment

This repository can deploy to a cloud server through GitHub Actions over SSH. The server builds Docker images locally and runs them with Docker Compose, so no image registry is required.

## Server Prerequisites

- Docker Engine with the `docker compose` plugin.
- SSH access from GitHub Actions using a private key.
- A deployment directory, default: `/opt/study-plan`.

## GitHub Secrets

- `SERVER_HOST`: cloud server IP or domain.
- `SERVER_USER`: SSH user, default `root` if omitted.
- `SERVER_PORT`: SSH port, default `22` if omitted.
- `SERVER_SSH_KEY`: private key used by GitHub Actions to SSH into the server.
- `DEPLOY_PATH`: server path for the checked out app, default `/opt/study-plan` if omitted.
- `PRODUCTION_ENV`: contents of the server `.env.production` file.

## Minimal `PRODUCTION_ENV`

```env
APP_ENV=production
PORT=8080
DB_PATH=/data/study_plan.db
JWT_SECRET=<strong-random-secret>
JWT_EXPIRE_HOURS=168
WECHAT_APPID=<mini-program-appid>
WECHAT_SECRET=<mini-program-secret>
WECHAT_LOGIN_MOCK=false
PHONE_BINDING_REQUIRED=false
ADMIN_USERNAME=<admin-username>
ADMIN_PASSWORD=<initial-admin-password>
AI_KEY_ENCRYPTION_SECRET=<strong-random-server-side-secret>
AVATAR_STORAGE=minio
AVATAR_BASE_URL=https://assets.example.com/study-plan-assets
MINIO_ROOT_USER=<minio-admin-user>
MINIO_ROOT_PASSWORD=<minio-strong-password>
MINIO_BUCKET=study-plan-assets
ARCHIVE_ENABLED=false
ARCHIVE_DRIVER=mysql
ARCHIVE_DSN=
ARCHIVE_INTERVAL_MINUTES=5
ARCHIVE_TABLES=users,plans,daily_tasks,checkins,study_sessions,slack_records
BACKEND_PORT=8080
FRONTEND_PORT=80
ADMIN_PORT=8081
MINIO_API_PORT=9000
MINIO_CONSOLE_PORT=9001
FRONTEND_API_BASE=/api
ADMIN_API_BASE=
```

## Published Services

- Frontend H5: `http://<server>:${FRONTEND_PORT}`.
- Backend API: `http://<server>:${BACKEND_PORT}` and also proxied from frontend/admin `/api`.
- Admin console: `http://<server>:${ADMIN_PORT}`.
- MinIO API: `http://<server>:${MINIO_API_PORT}`.
- MinIO console: `http://<server>:${MINIO_CONSOLE_PORT}`.
- Container names: `study-plan-backend`, `study-plan-frontend`, `study-plan-admin`, `study-plan-minio`, `study-plan-minio-init`.

For a production domain, put Nginx or a cloud load balancer in front of these ports and enable HTTPS. The backend SQLite database is stored in the Docker volume `study-plan-data` at `/data/study_plan.db` inside the backend container. MinIO object data is stored in the Docker volume `study-plan-minio-data`.

The deployment creates the bucket from `MINIO_BUCKET` and sets anonymous download permission. The current app stores avatar URLs; upload workflow still needs to place files into MinIO and then save the resulting public URL through the avatar API.

After deployment, configure AI through the admin console with provider `siliconflow`, pinned Base URL `https://api.siliconflow.cn/v1`, and recommended model `deepseek-ai/DeepSeek-V3.2`. SiliconFlow receives OpenAI-compatible `POST /chat/completions` requests; keep `AI_KEY_ENCRYPTION_SECRET` stable so stored API keys remain decryptable.

## Manual Server Deploy

```bash
cd /opt/study-plan
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build --remove-orphans
```
