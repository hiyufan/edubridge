# 后端开发指南

## 技术栈

| 组件 | 技术 |
|------|------|
| 框架 | Gin (github.com/gin-gonic/gin) |
| HTTP 客户端 | Resty (github.com/go-resty/resty/v2) |
| 解析 | goquery (github.com/PuerkitoBio/goquery) |
| 鉴权 | golang-jwt/jwt |
| 并发 | Goroutines + sync.Mutex |

## 项目结构

```
internal/
├── handler/          # HTTP 接口处理层
│   ├── auth.go       # 登录、Token 刷新、登出
│   ├── captcha.go    # 验证码获取
│   ├── schedule.go   # 课表
│   └── score.go      # 成绩
├── middleware/
│   └── auth.go       # JWT 鉴权中间件
├── model/
│   ├── schedule.go   # 课表数据结构
│   └── score.go      # 成绩数据结构
└── service/
    └── jw.go         # 教务系统爬虫核心逻辑

pkg/response/
└── response.go       # 统一响应封装

config/
└── config.go         # 环境变量读取
```

## 核心组件

### JwService（单例模式）

`jw.go` 中的 `JwService` 是整个后端的核心，以单例模式运行，管理所有与教务系统的 HTTP 会话。

```go
// 获取实例
svc := service.GetJwService()
```

**主要职责：**
- 维护与教务系统的 Session（Cookie 共享）
- 登录、课表、成绩的抓取逻辑
- 会话缓存与自动清理

**数据结构：**

```go
type JwService struct {
    sessions      map[string]*Session   // sessionId -> Session
    mu            sync.RWMutex
    scheduleCache map[string]*ScheduleCache
}

type Session struct {
    Client     *resty.Client  // Resty 客户端（用于课表/成绩）
    HttpClient *http.Client   // 标准库 Client（用于验证码/登录）
    CookieJar  *jar           // 共享 Cookie
    UID        string
    ExpireTime time.Time
    ScoreCache *ScoreCache
}
```

### 会话生命周期

- Session 有效期 **30 分钟**
- 后台 Goroutine 每分钟清理过期会话
- Session 失效后自动重新创建

### Token 机制

采用双 Token 方案：

| Token | 有效期 | 存储方式 | 用途 |
|-------|--------|----------|------|
| Access Token | 2 小时 | localStorage | API 请求认证 |
| Refresh Token | 30 天 | HttpOnly Cookie | 续期 Access Token |

## Handler 层规范

每个 Handler 遵循以下模式：

```go
func (h *Handler) Handle(c *gin.Context) {
    var req model.Request
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, "参数错误")
        return
    }

    data, err := svc.DoSomething(req)
    if err != nil {
        response.ErrorWithCode(c, http.StatusInternalServerError, err.Error(), "")
        return
    }

    response.Success(c, data)
}
```

### 响应格式

```go
// 成功
response.Success(c, data)

// 成功带 Token（登录）
response.SuccessWithToken(c, token, uid, expiresIn)

// 业务错误（登录失败、验证码错误等）
response.ErrorWithCode(c, http.StatusUnauthorized, "错误信息", "")

// HTTP 错误（参数缺失等）
response.Error(c, http.StatusBadRequest, "缺少必要参数")
```

## 中间件

### JWT 鉴权

`middleware/auth.go` 导出 `AuthRequired()` 中间件，使用方法：

```go
protected := api.Group("")
protected.Use(authMiddleware.AuthRequired())
```

中间件从 Token 中解析出 `uid` 和 `sessionId`，存入 Gin Context 供后续 Handler 使用。

### 限流

内嵌在 `main.go` 中的匿名中间件，实现简单的 IP 级别限流：

- 登录接口：5 分钟内最多 **10 次**
- 验证码接口：5 分钟内最多 **100 次**

## 开发规范

### 新增接口

1. 在 `model/` 中定义请求/响应结构体
2. 在 `handler/` 中实现处理函数
3. 在 `main.go` 的路由表中注册

### 教务系统对接

所有对教务系统的请求都通过 `Session.Client`（Resty）或 `Session.HttpClient`（标准库）发起，已自动处理：
- Cookie 共享（验证码和登录共用同一 Cookie Jar）
- Keep-Alive 连接
- 自动重定向（最多 10 次）

### 注意事项

- **不要** 直接使用 `net/http.DefaultClient`，会丢失 Cookie 共享
- Session 验证用 `checkSession()`，不要直接读缓存
- 教务系统返回的 HTML 使用 `goquery` 解析，不要用正则硬解析

## 测试

```bash
go test ./...
```

## 构建

```bash
go build -o jww-server main.go
```
