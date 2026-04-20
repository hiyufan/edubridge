# jww.p — 教务系统中间件

为高校教务系统打造的 Go 后端 + Vue 前端中间件，提供登录认证、课表查询、成绩查询等服务。开箱即用，部署简单。

> ⚠️ 本项目仅供学习与个人研究使用，请勿用于任何商业活动或大规模自动化操作。

## 功能特性

- 🔐 **安全登录** — 验证码 + JWT 双 Token 认证，支持 Token 自动刷新
- 📅 **课表查询** — 按周切换，自动识别当前周，支持全学期课表
- 📊 **成绩查询** — 按学期筛选，显示学分、绩点等详细数据
- 🧭 **今日课表** — 首页快速查看当天课程
- 📱 **响应式设计** — 支持手机和桌面端

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
| `/api/score` | GET | 成绩（参数: `semester`） |
| `/api/score/semesters` | GET | 可选学期列表 |

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

## 相关文档

- [docs/frontend/guide.md](docs/frontend/guide.md) — 前端开发指南
- [docs/frontend/api.md](docs/frontend/api.md) — 前端 API 参考
- [docs/backend/guide.md](docs/backend/guide.md) — 后端开发指南
- [docs/backend/api.md](docs/backend/api.md) — 后端 API 参考

## License

MIT
