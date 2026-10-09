# 后端开发指南

## 分层

```
main.go                 组装各组件，启动 HTTP 服务和后台任务（无业务逻辑）
cmd/fakejw/             本地开发用的模拟教务系统
internal/
├── config/             从环境变量读取并校验配置
├── model/              各层共享的纯数据类型
├── jwclient/           教务系统客户端：登录、验证码、课表、成绩、页面解析（无状态，不碰 Redis）
│   └── jwtest/         模拟教务系统（测试与 cmd/fakejw 共用）
├── timetable/          课表领域逻辑（纯函数）：作息时间、教学周、变动比对、冲突检测、iCal
├── grades/             成绩领域逻辑（纯函数）：新成绩比对、GPA 统计
├── store/              所有 Redis 读写，key 命名集中在 keys.go
├── session/            教务登录会话：登录前按 sessionId、登录后按学号，cookie 持久化与恢复
├── notify/             通知渠道（Webhook/PushPlus/邮件/群机器人）与“最近动态”
├── academic/           业务用例：查询（带缓存）、与快照比对并通知、处理教务登录失效
├── monitor/            后台调度：定时保活、定时完整检查（业务逻辑在 academic）
├── reminder/           上课提醒调度
├── auth/               本系统的 JWT 与 refresh token 轮换
├── api/                HTTP 路由、参数解析、统一响应与错误映射
└── testenv/            集成测试用：一行组装整套服务
```

依赖只能自上而下：`api → academic / auth / notify / reminder / monitor → session / store → jwclient / timetable / grades → model`。
没有全局变量和单例，所有依赖在 `main.go` 里通过构造函数传入。

## 关键流程

**登录**：`GET /api/captcha` 用前端的 sessionId 在 `session.Manager` 中建一个未登录的 `jwclient.Client` 并取验证码；
`POST /api/auth/login` 用同一个 Client 登录，成功后会话转为按学号保存，cookie 写入 Redis，用户加入后台监控，
再由 `auth` 签发 JWT（只包含学号）。

**教务登录态**：业务请求按学号取会话，内存没有时从 Redis 恢复。任何请求发现被踢回登录页都会返回
`jwclient.ErrSessionExpired`，由 `academic` 统一处理：清理登录态、停止监控、推送一次 `session-expired`；
API 返回 `401 + JW_SESSION_EXPIRED`，前端回到登录页。

**课表/成绩监控**：`monitor` 每 `MONITOR_KEEPALIVE_INTERVAL` 对每个用户保活一次，每 `MONITOR_CHECK_INTERVAL`
完整拉取课表和成绩。`academic` 把结果与 Redis 中的快照比对：
- 课表按“某周某节课”比对，只看当前周及以后；任一周拉取失败（不完整）时不比对，避免把缺的周误判为停课；
- 成绩第一次完整拉取只建立基线，之后新出现或被修改的成绩才通知。

**上课提醒**：`reminder` 每 30 秒按北京时间检查一次，基于课表快照发送；同一条提醒用 Redis `SETNX` 去重。

## 本地开发

```bash
redis-server &                                   # 或使用已有 Redis
go run ./cmd/fakejw &                            # 模拟教务系统 :8081，账号 2024001 / pw，验证码 1234
JWT_SECRET=dev-a JWT_REFRESH_SECRET=dev-b JW_URL=http://localhost:8081 go run .
cd frontend && npm run dev                       # http://localhost:5173
```

## 测试

```bash
go test -race ./...
```

- `jwclient` 对模拟教务系统测试请求与解析；`timetable`、`grades` 是纯函数单元测试；
- `monitor`、`reminder`、`api` 通过 `testenv` 组装整套服务（模拟教务系统 + miniredis + 记录通知的 Webhook）做集成测试，
  覆盖登录、课表变动、成绩发布、登录失效、服务重启恢复、HTTP 全流程。

## 新增功能时

- 和教务系统交互 → `jwclient`（附带 `jwtest` 模拟与测试）；
- 只依赖数据的计算 → `timetable` / `grades`（纯函数 + 单元测试）；
- 需要持久化 → 在 `store` 加方法，key 写进 `keys.go`；
- 组合以上完成一个用例 → `academic`（或独立的小服务包）；
- HTTP 层只做参数解析和调用，错误交给 `api.fail` 统一映射。
