## 1. Production Configuration

- [x] 1.1 Add production startup validation for required environment variables
- [x] 1.2 Make mock login unavailable unless `WECHAT_LOGIN_MOCK=true`
- [x] 1.3 Add documented `.env.production.example`

## 2. Real WeChat Login

- [x] 2.1 Implement `jscode2session` request using `WECHAT_APPID` and `WECHAT_SECRET`
- [x] 2.2 Handle WeChat API errors with clear API responses
- [x] 2.3 Keep local mock login for development only
- [x] 2.4 Implement optional avatar update after user selection/authorization
- [x] 2.5 Implement required phone number binding after user authorization
- [x] 2.6 Block study features until phone number is verified and bound
- [ ] 2.7 Verify WeChat phone-number capability cost/quota in the WeChat console before release
- [x] 2.8 Store avatar as URL or object-storage key, not database binary data
- [x] 2.9 Support object-storage compatible avatar storage configuration for COS or self-hosted MinIO

## 3. Subscription Messages

- [x] 3.1 Add notification template configuration
- [x] 3.2 Persist notification delivery attempts and failures
- [x] 3.3 Implement sender for study start, completion, 23:30 decision, and missed check-in reminders
- [x] 3.4 Only send reminders when user has subscribed to the template and admin has enabled that reminder type

## 4. Logs, Backups, And Archive Sync

- [x] 4.1 Add structured request and error logs
- [x] 4.2 Add SQLite backup command or script
- [x] 4.3 Document restore procedure
- [x] 4.4 Add archive sync configuration for optional MySQL archival
- [x] 4.5 Implement archive job that copies selected SQLite tables to MySQL when enabled
- [x] 4.6 Log archive failures without blocking normal app operations
- [x] 4.7 Default enabled archive interval to 5 minutes
- [x] 4.8 Document daily SQLite backup and manual pre-deploy backup command
- [x] 4.9 Document same-server Docker MinIO deployment option on Tencent Cloud

## 5. Verification

- [x] 5.1 Verify backend build
- [x] 5.2 Verify mini program build with production `VITE_API_BASE`
- [x] 5.3 Verify local mock-mode login and local production-mode configuration checks
- [x] 5.4 Update Tencent Cloud deployment guide
