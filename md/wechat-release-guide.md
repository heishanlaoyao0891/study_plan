# 微信小程序发布指南

## 需要准备

1. 微信小程序账号

个人账号即可。当前项目没有真实支付、押金池或企业付款能力，不需要微信支付商户号。

2. 小程序 AppID / AppSecret

在微信公众平台后台获取，用于后端真实微信登录。

3. 公网服务器

用于部署 Go 后端。需要公网 IP 和可访问的域名。

4. 已备案域名

微信小程序正式发布要求接口域名使用 HTTPS。国内服务器通常需要域名备案。

5. HTTPS 证书

可以使用云厂商免费证书或 Let's Encrypt。

6. 后端生产配置

参考 `backend/.env.example` 创建生产 `.env`：

```env
PORT=8080
DB_PATH=/var/lib/study-plan/study_plan.db
JWT_SECRET=change-this-to-a-strong-random-secret
WECHAT_APPID=your-mini-program-appid
WECHAT_SECRET=your-mini-program-secret
WECHAT_LOGIN_MOCK=false
AI_PROVIDER=mock
AI_API_KEY=
AI_BASE_URL=
```

## 发布步骤

1. 注册并配置小程序

在微信公众平台注册小程序，完成名称、头像、简介、服务类目等基础信息配置。

2. 准备服务器环境

在云服务器上准备运行目录和数据目录：

```bash
mkdir -p /var/lib/study-plan
```

可以选择在服务器安装 Go 后编译，也可以本地交叉编译后上传二进制。

3. 编译后端

```bash
cd backend
go build -o study_plan_backend .
```

4. 配置后端环境变量

在后端运行目录创建 `.env`，填入真实配置。生产环境必须设置：

```env
WECHAT_LOGIN_MOCK=false
WECHAT_APPID=your-mini-program-appid
WECHAT_SECRET=your-mini-program-secret
JWT_SECRET=strong-random-secret
```

5. 启动后端

推荐使用 `systemd` 管理后端进程，保证服务器重启或进程异常退出后自动恢复。

6. 配置 Nginx 反向代理

参考 `deploy/nginx.study-plan.conf`，将示例里的：

```nginx
server_name example.com;
```

替换为真实域名，并配置 HTTPS 证书。生产小程序必须通过 HTTPS 访问后端接口。

7. 配置微信后台合法域名

在微信公众平台后台进入：

```text
开发管理 -> 开发设置 -> 服务器域名
```

将后端 HTTPS 域名添加到 `request 合法域名`，例如：

```text
https://api.example.com
```

8. 配置前端生产 API 地址

构建小程序前设置生产 API 地址：

```bash
cd frontend
VITE_API_BASE=https://api.example.com npm run build:mp-weixin
```

Windows PowerShell 可以使用：

```powershell
$env:VITE_API_BASE="https://api.example.com"
npm run build:mp-weixin
```

9. 导入微信开发者工具

打开微信开发者工具，导入构建产物目录：

```text
frontend/dist/build/mp-weixin
```

10. 真机预览测试

使用微信开发者工具生成预览二维码，在手机上验证核心流程：

- 微信登录
- 创建计划
- AI 生成计划
- 今日学习计时
- 完成任务与打卡
- 躺平币余额和记录
- 统计页
- 管理员页面

11. 上传代码

在微信开发者工具点击 `上传`，填写版本号和版本说明。

12. 提交审核

在微信公众平台提交审核。审核通过后，点击发布。

## 上线前检查

- 确认 `WECHAT_LOGIN_MOCK=false`。
- 确认 `JWT_SECRET` 已替换为强随机字符串。
- 确认小程序请求域名是 HTTPS 且已加入微信后台合法域名。
- 确认前端构建时 `VITE_API_BASE` 指向生产后端域名。
- 确认数据库文件目录可写，并有备份策略。
- 当前 AI 生成仍是 mock provider。如果不接真实 AI，建议在审核前弱化或隐藏真实 AI 承诺。
- 当前微信订阅消息是提醒事件队列和占位订阅接口。若要真实推送，需要在微信后台申请订阅消息模板，并接入模板发送逻辑。

## 规模建议

SQLite 足够支撑个人和朋友范围使用。如果后续公开推广，建议迁移到 MySQL 或 PostgreSQL，并补充日志、备份、监控和限流。
