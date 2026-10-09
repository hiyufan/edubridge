# jww.p — 教务系统中间件

为高校教务系统打造的 Go 后端 + Vue 前端中间件，提供登录认证、课表查询、成绩查询、GPA 趋势分析、课程冲突检测、iCal 订阅、Webhook 通知等服务。开箱即用，部署简单。

> ⚠️ 本项目仅供学习与个人研究使用，请勿用于任何商业活动或大规模自动化操作。

## 功能特性

- 🔐 **安全登录** — 验证码 + JWT 双 Token 认证，支持 Token 自动刷新；教务登录态持久化到 Redis，服务重启不掉线
- 👀 **课表监控** — 后台定时检查课表，发现加课、减课/停课、换教室时推送通知，并保持教务登录不过期
- 🎓 **成绩发布提醒** — 新成绩出来（或成绩被修改）时推送课程、成绩、绩点
- ⏰ **上课提醒** — 每天定时推送当天/明天的课，或每节课开始前 N 分钟提醒
- 📋 **监控状态** — 「我的」页面显示监控是否运行、上次检查时间和最近动态
- 📅 **课表查询** — 按周切换，自动识别当前周，支持全学期课表
- ⚠️ **冲突检测** — 自动检测同一时段重复安排的课程
- 📊 **成绩查询** — 按学期筛选，显示学分、绩点等详细数据
- 📈 **GPA 趋势图** — 原生 Canvas 绘制，贝塞尔曲线+渐变填充+数据点发光效果
- 🧭 **今日课表** — 首页快速查看当天课程
- 📱 **响应式设计** — 支持手机和桌面端
- 🗓️ **iCal 订阅** — 生成标准 iCalendar 格式，可导入 Google Calendar/Apple Calendar/Outlook
- 🔔 **多渠道通知** — 课表变动、教务登录失效时推送到微信（PushPlus）、邮箱、企业微信/钉钉/飞书群机器人或自定义 Webhook，可同时开启多个

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.21+ / Gin / golang-jwt |
| 前端 | Vue 3 (Composition API) / Vite / Pinia / Vue Router |
| UI | Tailwind CSS / Element Plus |
| 通信 | REST API + JWT（Access Token + Refresh Token） |

## 快速开始

### 前置要求

- Go 1.21+
- Node.js 18+
- 目标教务系统地址（默认: `https://jw.fzrjxy.com`）

### 克隆项目

```bash
git clone https://github.com/cyfстрой/jww.p.git
cd jww.p
```

### 配置环境变量

```bash
# Linux / macOS
export JWT_SECRET="your-access-token-secret"
export JWT_REFRESH_SECRET="your-refresh-token-secret"
export PORT="3000"

# Windows (PowerShell)
$env:JWT_SECRET="your-access-token-secret"
$env:JWT_REFRESH_SECRET="your-refresh-token-secret"
$env:PORT="3000"
```

> JWT 密钥请使用足够长的随机字符串，Access Token 密钥和 Refresh Token 密钥不可相同。

### 启动后端

```bash
go mod tidy
go run main.go
```

### 启动前端（开发模式）

```bash
cd frontend
npm install
npm run dev
```

访问 `http://localhost:5173` 即可使用。

## 项目结构

```
jww.p/
├── main.go                    # Go 服务入口
├── go.mod / go.sum            # Go 依赖
├── config/
│   └── config.go              # 环境变量读取
├── internal/
│   ├── handler/              # HTTP 接口处理
│   │   ├── auth.go           # 登录 / 刷新 Token / 登出
│   │   ├── captcha.go        # 验证码获取
│   │   ├── schedule.go        # 课表查询
│   │   └── score.go           # 成绩查询
│   ├── middleware/
│   │   └── auth.go            # JWT 鉴权中间件
│   ├── model/
│   │   ├── schedule.go        # 课表数据模型
│   │   └── score.go           # 成绩数据模型
│   └── service/
│       └── jw.go              # 教务系统爬虫核心逻辑
├── pkg/response/
│   └── response.go            # 统一响应格式
└── frontend/                  # Vue 3 前端
    ├── src/
    │   ├── views/             # 页面组件
    │   │   ├── Login.vue      # 登录页
    │   │   ├── Schedule.vue   # 课表页
    │   │   ├── Score.vue      # 成绩页
    │   │   ├── Today.vue      # 今日课表
    │   │   └── Profile.vue    # 个人中心
    │   ├── stores/            # Pinia 状态管理
    │   ├── utils/             # 工具函数
    │   ├── router/            # 路由配置
    │   └── layout/             # 页面布局
    └── dist/                  # 构建产物（静态嵌入）
```

## API 接口

基础路径: `/api`

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/captcha` | GET | 获取验证码图片 + sessionId |
| `/api/auth/login` | POST | 登录 |
| `/api/auth/refresh` | POST | 刷新 Access Token |
| `/api/auth/logout` | POST | 登出 |
| `/api/auth/me` | GET | 获取当前用户信息 |
| `/api/schedule` | GET | 单周课表（参数: `week`） |
| `/api/schedule/full` | GET | 全学期课表（参数: `maxWeek`） |
| `/api/schedule/conflicts` | GET | 课程冲突检测 |
| `/api/schedule/ical` | GET | iCal 日历订阅 |
| `/api/score` | GET | 成绩（参数: `semester`） |
| `/api/score/stats` | GET | 成绩统计（GPA、学分、挂科等） |
| `/api/score/semesters` | GET | 可选学期列表 |
| `/api/schedule/diff` | GET | 最近一次检测到的课表变动 |
| `/api/monitor/status` | GET | 监控状态、上次检查/保活时间、最近动态 |
| `/api/notify/reminder` | GET/PUT | 查询/保存上课提醒设置 |
| `/api/webhook/register` | POST | 注册 Webhook（`url`, `secret`） |
| `/api/webhook/info` | GET | 查询已注册的 Webhook |
| `/api/webhook/trigger` | POST | 把最近一次课表变动重新推送一次 |
| `/api/notify/channels` | GET/PUT | 查询/保存通知渠道（微信、邮箱、群机器人） |
| `/api/notify/test` | POST | 向所有已开启的渠道发送测试通知，返回各渠道结果 |

### 课表监控与 Webhook

登录后用户自动进入监控列表：每 `MONITOR_KEEPALIVE_INTERVAL` 访问一次教务系统保持登录，每 `MONITOR_CHECK_INTERVAL` 拉取一次完整课表并与上次比对（只比对当前周及以后）。部分周拉取失败时本次不比对，避免误报。教务登录失效时停止监控并推送 `session-expired`，用户重新登录后自动恢复。

Webhook 请求为 `POST application/json`，请求头 `X-JWW-Event` 为事件类型，配置了密钥时带 `X-Hub-Signature-256: sha256=<HMAC-SHA256(body)>`：

```json
{
  "event": "schedule-diff",
  "text": "课表有变动：\n【加课】\n+ 大学物理 周三 第1-2节 @B202（第4周）\n【减课/停课】\n- 高等数学 周一 第1-2节 @A101（第5、6周）",
  "data": { "added": [...], "removed": [...], "changed": [...], "detectedAt": "..." },
  "time": "2026-10-09T08:00:00+08:00"
}
```

`event` 取值：`schedule-diff`（课表变动）、`score-new`（新成绩）、`reminder`（上课提醒）、`session-expired`（需要重新登录）、`test`（测试）。`text` 可直接转发到微信/群机器人。

### 登录请求示例

```bash
curl -X POST http://localhost:3000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "2405030544",
    "password": "your-password",
    "captcha": "abcd",
    "loginType": "xsxh",
    "sessionId": "your-session-id"
  }'
```

### 响应格式

```json
{
  "status": 1,
  "info": "登录成功",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "uid": "2405030544"
}
```

错误时 `status: 0`，`info` 包含错误信息。

## 生产部署

### 构建

```bash
# 构建后端
go build -o jww-server main.go

# 构建前端（可选，已嵌入 dist）
cd frontend && npm run build
```

### 运行

```bash
JWT_SECRET="your-secret" JWT_REFRESH_SECRET="your-refresh-secret" PORT=3000 ./jww-server
```

### Nginx 配置

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 前端静态文件
    location / {
        root /path/to/jww.p/frontend/dist;
        try_files $uri $uri/ /index.html;
    }

    # API 反向代理
    location /api {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `JWT_SECRET` | ✅ | — | Access Token 签名密钥 |
| `JWT_REFRESH_SECRET` | ✅ | — | Refresh Token 签名密钥（不可与 JWT_SECRET 相同） |
| `PORT` | ❌ | `3000` | 服务端口 |
| `MONITOR_KEEPALIVE_INTERVAL` | ❌ | `10m` | 保持教务登录的访问间隔，需小于学校会话超时时间 |
| `MONITOR_CHECK_INTERVAL` | ❌ | `1h` | 完整拉取课表比对变动的间隔 |
| `SMTP_HOST` | ❌ | — | 发信邮箱 SMTP 服务器（如 `smtp.qq.com`），不配置则邮件通知不可用 |
| `SMTP_PORT` | ❌ | `465` | 465 使用 SSL，587 等使用 STARTTLS |
| `SMTP_USER` | ❌ | — | 发信账号 |
| `SMTP_PASSWORD` | ❌ | — | 授权码（QQ/163 邮箱需在邮箱设置中开启 SMTP 并生成授权码） |
| `SMTP_FROM` | ❌ | 同 `SMTP_USER` | 发件人地址 |

## 相关文档

- [docs/frontend/guide.md](docs/frontend/guide.md) — 前端开发指南
- [docs/frontend/api.md](docs/frontend/api.md) — 前端 API 参考
- [docs/backend/guide.md](docs/backend/guide.md) — 后端开发指南
- [docs/backend/api.md](docs/backend/api.md) — 后端 API 参考

## License

MIT
