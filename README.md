# EduBridge — 教务系统中间件

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
| 后端 | Go 1.21+ / Gin / golang-jwt / Redis |
| 前端 | Vue 3 (Composition API) / Vite / Pinia / Vue Router / Element Plus |
| 通信 | REST API + JWT（Access Token + Refresh Token） |

## 快速开始

需要 Go 1.21+、Node.js 18+、Redis。

```bash
git clone https://github.com/hiyufan/edubridge.git
cd edubridge
cp .env.example .env          # 修改 JWT 密钥和 Redis 配置
go run .                      # 后端 :3000
cd frontend && npm install && npm run dev   # 前端 http://localhost:5173（/api 代理到 :3000）
```

**不连学校服务器也能开发**：`go run ./cmd/fakejw` 启动一个模拟教务系统（:8081，账号 `2024001` / 密码 `pw` / 验证码 `1234`），
再用 `JW_URL=http://localhost:8081 go run .` 启动后端即可。

## 项目结构

```
main.go            组装各组件并启动服务
cmd/fakejw/        本地开发用的模拟教务系统
internal/
  jwclient/        教务系统客户端（请求 + 页面解析）
  timetable/       课表领域逻辑：教学周、作息、变动比对、冲突、iCal（纯函数）
  grades/          成绩领域逻辑：新成绩比对、GPA 统计（纯函数）
  store/           Redis 读写
  session/         教务登录会话管理
  academic/        业务用例：查询、快照比对、登录失效处理
  monitor/         后台保活与定时检查
  reminder/        上课提醒
  notify/          通知渠道与最近动态
  auth/            JWT 与 refresh token
  api/             HTTP 路由与统一错误处理
  config/ model/ testenv/
frontend/src/
  views/           页面（只负责布局）
  components/      schedule/、profile/ 下的页面组件
  stores/ utils/ composables/ styles/
```

分层和关键流程见 [docs/backend/guide.md](docs/backend/guide.md)。

## 测试

```bash
go test -race ./...           # 后端（含模拟教务系统的端到端测试）
cd frontend && npm run build  # 前端构建检查
```

## 部署

```bash
cp .env.example .env   # 填写 JWT 密钥、Redis、（可选）SMTP
docker compose up -d   # 或：go build -o jww . && ./jww
```

前端 `npm run build` 后由 Nginx 提供静态文件，并把 `/api` 反向代理到后端：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        root /path/to/frontend/dist;
        try_files $uri $uri/ /index.html;
    }

    location /api {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `JWT_SECRET` | ✅ | — | Access Token 签名密钥 |
| `JWT_REFRESH_SECRET` | ✅ | — | Refresh Token 签名密钥（不可与 JWT_SECRET 相同） |
| `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` | ❌ | `localhost` / `6379` / 空 | Redis 连接 |
| `PORT` | ❌ | `3000` | 服务端口 |
| `ALLOWED_ORIGIN` | ❌ | `http://localhost:5173` | 允许跨域的前端地址 |
| `SECURE_COOKIE` | ❌ | `false` | HTTPS 部署时设为 `true` |
| `JW_URL` | ❌ | `https://jw.fzrjxy.com` | 教务系统地址 |
| `MONITOR_KEEPALIVE_INTERVAL` | ❌ | `10m` | 保持教务登录的访问间隔，需小于学校会话超时时间 |
| `MONITOR_CHECK_INTERVAL` | ❌ | `1h` | 完整拉取课表和成绩比对变动的间隔 |
| `SMTP_HOST` | ❌ | — | 发信邮箱 SMTP 服务器（如 `smtp.qq.com`），不配置则邮件通知不可用 |
| `SMTP_PORT` | ❌ | `465` | 465 使用 SSL，587 等使用 STARTTLS |
| `SMTP_USER` / `SMTP_PASSWORD` | ❌ | — | 发信账号与授权码（QQ/163 邮箱需在设置中开启 SMTP 并生成授权码） |
| `SMTP_FROM` | ❌ | 同 `SMTP_USER` | 发件人地址 |

## 相关文档

- [docs/api.md](docs/api.md) — API 参考（含 Webhook 格式）
- [docs/backend/guide.md](docs/backend/guide.md) — 后端分层与开发指南
- [docs/frontend/guide.md](docs/frontend/guide.md) — 前端开发指南

## License

MIT
