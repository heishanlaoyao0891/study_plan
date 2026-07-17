## Context

智能学习打卡小程序。Go 后端 + 微信小程序前端 + SQLite。核心特色：AI 生成计划、计时打卡双模式、躺平奖励机制、灵活的延期系统。

## Goals / Non-Goals

**Goals:**
- AI 根据用户描述生成每日学习计划，用户可编辑；也支持手动创建
- 多学习计划并行，每个计划独立时间段和每周时长目标
- 固定时段自动计时 + 临时开始/结束按钮
- 完成打卡赚取躺平时长
- 灵活延期：推迟到某天、整体平移、暂停计划
- 微信订阅消息多场景推送
- 管理员唯一，可永久或限时封禁用户
- 管理员后台配置躺平比例（全局/按用户）
- 深度统计：时长分布、躺平日志、周/月报告

**Non-Goals:**
- 好友关系/排行榜（可能后续版本）
- 多人协作学习
- 视频/直播教学

## Decisions

| 决策 | 选择 | 理由 |
|------|------|------|
| Web 框架 | Gin | Go 生态最成熟 |
| 数据库 | SQLite (mattn/go-sqlite3) | 单文件部署，零运维 |
| API 风格 | RESTful JSON | 小程序友好 |
| 登录方案 | 微信登录 + JWT | openid 做标识 |
| AI 接口 | GPT/Claude API | 结构化 JSON 输出，直接生成任务列表 |
| 提醒方案 | 微信订阅消息 | 小程序原生 |
| 定时任务 | robfig/cron | 进程内调度 |
| 计时方案 | 后端记录 start/end 时间戳，前端轮询或 ws 同步 | 精确到秒 |

## Data Model

```
User
  id              INTEGER PK
  openid          TEXT UNIQUE
  nickname        TEXT
  avatar_url      TEXT
  weekly_hours    INTEGER DEFAULT 0   // 每周目标时长
  slack_balance   INTEGER DEFAULT 0   // 躺平剩余分钟数
  role            TEXT DEFAULT 'user'  // user / admin
  banned_until    DATETIME             // NULL=未封禁, 过去时间=已解封
  banned_reason   TEXT
  created_at      DATETIME

StudyPlan
  id              INTEGER PK
  user_id         INTEGER FK -> User.id
  title           TEXT                 // "学习Go语言"
  description     TEXT
  status          TEXT                 // active / paused / archived
  weekly_target_hours INTEGER          // 每周目标时长
  start_date      DATE
  end_date        DATE
  ai_generated    BOOLEAN DEFAULT 0
  is_shared       BOOLEAN DEFAULT 0
  created_at      DATETIME

PlanMember
  id              INTEGER PK
  plan_id         INTEGER FK -> StudyPlan.id
  user_id         INTEGER FK -> User.id
  role            TEXT                 // owner / member
  joined_at       DATETIME
  UNIQUE(plan_id, user_id)

DailyTask
  id              INTEGER PK
  plan_id         INTEGER FK -> StudyPlan.id
  user_id         INTEGER FK -> User.id
  date            DATE
  title           TEXT                 // "Day1: 安装Go环境 + Hello World"
  description     TEXT
  scheduled_start TIME                 // 计划开始时间 (如 20:00)
  scheduled_end   TIME                 // 计划结束时间 (如 21:00)
  status          TEXT                 // pending / in_progress / completed / postponed / missed
  actual_start    DATETIME
  actual_end      DATETIME
  study_minutes   INTEGER              // 实际学习分钟数
  sort_order      INTEGER
  is_overtime     BOOLEAN DEFAULT 0   // 是否超出计划时长
  created_at      DATETIME

StudySession
  id              INTEGER PK
  task_id         INTEGER FK -> DailyTask.id
  user_id         INTEGER FK -> User.id
  start_time      DATETIME
  end_time        DATETIME
  duration_min    INTEGER
  note            TEXT
  created_at      DATETIME

Checkin
  id              INTEGER PK
  user_id         INTEGER FK -> User.id
  date            DATE
  plan_id         INTEGER FK -> StudyPlan.id
  completed       BOOLEAN
  created_at      DATETIME
  UNIQUE(user_id, date, plan_id)

PostponeRecord
  id              INTEGER PK
  task_id         INTEGER FK -> DailyTask.id
  postpone_type   TEXT          // to_date / shift_all / pause
  target_date     DATE          // 推迟到哪天
  target_time     TIME          // 推迟到几点
  reason          TEXT
  created_at      DATETIME

SlackRecord
  id              INTEGER PK
  user_id         INTEGER FK -> User.id
  start_time      DATETIME
  end_time        DATETIME
  duration_min    INTEGER
  activity        TEXT          // 躺平干啥（钓鱼/刷视频等）
  created_at      DATETIME

SlackConfig
  id              INTEGER PK
  user_id         INTEGER FK -> User.id  (NULL 表示全局默认)
  checkin_minutes   INTEGER     // 打卡奖励分钟数
  streak_bonus      INTEGER     // 连续打卡额外奖励
  quality_bonus     INTEGER     // 高质量完成额外奖励
  updated_by      INTEGER FK -> User.id  // 管理员
  created_at      DATETIME

WeeklyReport
  id              INTEGER PK
  user_id         INTEGER FK -> User.id
  year            INTEGER
  week            INTEGER
  total_study_min    INTEGER
  target_min         INTEGER
  completed_rate     REAL
  slack_min          INTEGER
  perfect_days       INTEGER
  created_at      DATETIME
```

### 登录拦截
```
用户登录 → 校验 banned_until
  ├ NULL → 正常登录
  └ 有值 → 判断 banned_until > now?
      ├ 是 → 拒绝登录，返回封禁原因和解封时间
      └ 否 → 自动清除 banned_until，正常登录
```

## API Design

```
// 认证
POST   /api/auth/login

// AI 生成计划
POST   /api/ai/generate-plan    // 描述学习目标 → AI 返回 DailyTask[]
POST   /api/ai/regenerate       // 重新生成某个计划
PUT    /api/ai/plan/:id/edit    // 用户编辑 AI 生成后的计划

// 学习计划
GET    /api/plans
POST   /api/plans                 // { title, schedule, member_ids? }
PUT    /api/plans/:id
DELETE /api/plans/:id
PUT    /api/plans/:id/pause
PUT    /api/plans/:id/resume
PUT    /api/plans/:id/shift     // 整体平移
POST   /api/plans/:id/invite    // 邀请加入组团
POST   /api/plans/:id/join      // 接受邀请


// 每日任务
GET    /api/tasks?date=
GET    /api/tasks/:id
PUT    /api/tasks/:id/start     // 开始学习
PUT    /api/tasks/:id/pause     // 暂停（处理事情后恢复）
PUT    /api/tasks/:id/resume    // 恢复
PUT    /api/tasks/:id/stop      // 结束学习
PUT    /api/tasks/:id/extend    // 延长学习
PUT    /api/tasks/:id/postpone  // 推迟到某天某时
PUT    /api/tasks/:id/makeup    // 补录结束时间
PUT    /api/tasks/:id/complete  // 标记任务完成

// 打卡
GET    /api/checkins?date=
POST   /api/checkins

// 躺平
POST   /api/slack/start         // 开始躺平
PUT    /api/slack/stop          // 结束躺平
GET    /api/slack/records       // 躺平日志
GET    /api/slack/balance       // 躺平余额

// 统计
GET    /api/stats/calendar?month=
GET    /api/stats/streak
GET    /api/stats/daily-distribution?date=
GET    /api/stats/weekly-report?year=&week=
GET    /api/stats/monthly-report?year=&month=
GET    /api/stats/slack-distribution?month=

// 管理员配置
GET    /api/admin/users
POST   /api/admin/users/:id/ban          // 封禁用户 {duration_hours: 0(永久) | 24 | 48, reason}
POST   /api/admin/users/:id/unban        // 解封用户
GET    /api/admin/slack-config
PUT    /api/admin/slack-config           // 全局默认
PUT    /api/admin/slack-config/:userId   // 用户定制

// 提醒
GET    /api/notifications/subscriptions
POST   /api/notifications/subscribe
DELETE /api/notifications/subscribe
```

## 关键业务流程

### AI 生成计划
```
用户输入 → POST /api/ai/generate-plan
  { goal: "学Go", hours_per_day: 2, start_date: "2026-07-20", skip_dates: ["2026-07-25"], available_time: "20:00-22:00" }
  ↓
  AI 返回 JSON: [{ date, title, description, scheduled_start, scheduled_end }, ...]
  ↓
  批量创建 DailyTask → 返回给用户可编辑
  ↓
  用户可手动增/删/改/调序
```

### 每日执行流程
```
到 scheduled_start → 微信提醒 "该学Go了"
  ├ [开始学习] → 记录 actual_start, 状态→in_progress
  ├ [推迟30分] → 推迟提醒
  ├ [推迟1小时]
  └ [今天延期] → 选择延期到哪天+几点 → 创建 PostponeRecord

学习中:
  [暂停] → 处理事情
  [恢复] → 继续计时
  [结束学习] → 记录 actual_end, 计算 study_minutes

到 scheduled_end 时:
  ├ 已完成今日任务 → 微信提醒 "今日计划完成"
  ├ 还想学 → [延长学习] → 继续计时
  └ 没完成 → 继续学

到 23:30:
  提醒用户做决策: 延到明天? / 继续学?
  如果忘记结束 → 后可以补录结束时间
```

## Risks / Trade-offs

- SQLite 并发写入性能有限 —— 个人/小团队场景足够
- AI API 调用有成本 —— 生成计划是一次性，成本可控
- 微信订阅消息是一次性的 —— 用户每天需主动订阅，这是微信限制
- 计时准确定依赖用户操作 —— 忘记开始/结束用补录机制兜底
