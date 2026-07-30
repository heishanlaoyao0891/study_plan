# 微信小程序发布指南

本项目的生产入口统一为 `https://slls.asia`：H5 使用 `/`，管理台使用 `/admin/`，后端 API 使用 `/api/`。小程序生产包禁止使用 IP、HTTP 或登录页临时覆盖 API 地址。

## 1. 发布前准备

- 微信小程序 AppID：`wx985c473e161501fc`。
- 后端生产配置已设置真实 `WECHAT_APPID`、`WECHAT_SECRET`，且 `WECHAT_LOGIN_MOCK=false`。
- `slls.asia` 与 `www.slls.asia` DNS 已解析到生产服务器。
- `https://slls.asia` 证书链有效，TLS 1.2/1.3 可用。
- 微信公众平台“开发管理 -> 开发设置 -> 服务器域名”的 request 合法域名包含 `https://slls.asia`。
- 发布前已备份 SQLite 和 MinIO 数据，并记录当前生产 Git revision。

## 2. 验证生产服务

```bash
curl --fail --show-error https://slls.asia/health
curl --fail --show-error --head https://slls.asia/
curl --fail --show-error --head https://slls.asia/admin/
```

如需排查容器，不开放新的公网端口；通过 SSH 检查 loopback：

```bash
curl --fail http://127.0.0.1:8080/health
docker compose --env-file .env.production -f docker-compose.prod.yml ps
docker logs --tail 200 study-plan-backend
```

## 3. 生成小程序发布包

```bash
cd frontend
npm ci
npm run release:mp-weixin
```

该命令会依次验证：

- `frontend/.env.production` 固定为 `VITE_API_BASE=https://slls.asia`。
- `VITE_ENABLE_DEV_LOGIN=false`。
- manifest AppID 与版本号有效。
- 前端测试和 TypeScript 检查通过。
- 生成 `frontend/dist/build/mp-weixin`。
- 构建产物包含正确 AppID 和生产域名。

当前候选版本为 `1.0.1`，versionCode 为 `101`。每次重新上传正式版本前都应递增这两个字段。

## 4. 开发者工具与真机验证

1. 微信开发者工具导入 `frontend/dist/build/mp-weixin`。
2. 确认项目 AppID 正确，且未勾选“开发环境不校验请求域名”作为正式验收条件。
3. 使用预览二维码在真机验证微信登录、创建计划、AI 生成、任务计时/打卡、统计、头像与重新登录。
4. 在服务端日志中确认请求 Host 为 `slls.asia`，没有向旧 IP 发起请求。

## 5. 上传、审核与发布

在微信开发者工具点击“上传”，版本号填写 `1.0.1`，建议版本说明：

```text
首个备案域名正式版：启用微信登录、学习计划、AI 任务拆解、计时打卡与学习统计。
```

上传后在微信公众平台完成隐私保护指引、服务类目、用户信息用途和审核材料，再提交审核。审核通过后仍需人工点击“发布”。逐项记录在 `docs/wechat-submission-checklist.md`。

## 6. 回滚

若部署后的 HTTPS/H5/API 检查失败，回退到上一 Git revision 后重新执行生产 Compose 部署。保留 `.env.production`、`study-plan-data` 和 `study-plan-minio-data`，不要通过删除 volume 回滚应用代码。

小程序已上传但未发布时可直接撤回审核或提交修复版本；已经发布时，在微信公众平台使用版本回退能力，并同步回退服务器到兼容版本。
