# API 参考

基础路径 `/api`。除标注「公开」的接口外，都需要请求头 `Authorization: Bearer <access token>`。

## 响应格式

```jsonc
// 成功
{ "status": 1, "data": { ... } }
// 失败
{ "status": 0, "info": "可直接展示给用户的说明", "code": "可选错误码" }
```

| HTTP 状态 | 含义 |
|---|---|
| 400 | 参数或设置不合法、教务系统拒绝登录（验证码/密码错误），`info` 说明原因 |
| 401 | 未登录或 access token 过期（前端应调用 `/auth/refresh`）；`code=JW_SESSION_EXPIRED` 表示**教务系统**登录已失效，需要重新输入验证码登录 |
| 404 | 数据不存在或已过期 |
| 429 | 请求过于频繁 |
| 502 | 教务系统暂时无法访问 |
| 500 | 服务器内部错误（详情只记录在服务端日志） |

## 认证

登录后返回 access token（2 小时），同时在 HttpOnly Cookie `refreshToken` 中写入 refresh token（30 天，每次刷新轮换）。
教务系统的登录态由服务端按学号保存在 Redis 中并定时保活，用户一般只在第一次登录时需要输入验证码。

| 接口 | 说明 |
|---|---|
| `GET /captcha`（公开） | 返回 `{status, data: "data:image/png;base64,...", sessionId}`；登录时带回 `sessionId` |
| `POST /auth/login`（公开） | 请求 `{sessionId, username, password, captcha, loginType?}`；返回 `{status, info, token, uid, expiresIn}` |
| `POST /auth/refresh`（公开） | 凭 Cookie 换新 token：`{status, token, expiresIn}` |
| `POST /auth/logout`（公开） | 作废 refresh token，清除教务登录态并停止后台监控 |
| `GET /auth/me` | `{uid}` |

## 课表

| 接口 | 说明 |
|---|---|
| `GET /schedule?week=N` | 单周课表；不传 `week` 返回真实当前周。`data`: `{semester, className, studentName, week, currentWeek, semesterStart, courses[]}` |
| `GET /schedule/full?maxWeek=N` | 全学期课表（缓存 5 分钟），`courses[].weeks` 为上课周次 |
| `GET /schedule/conflicts` | 时间冲突：`[{courseA, courseB, conflictWeeks}]` |
| `GET /schedule/diff` | 最近一次检测到的变动：`{added[], removed[], changed[{old,new}], detectedAt}`，没有为 `null` |
| `GET /schedule/ical` | 下载 iCalendar 文件 |
| `POST /schedule/ical/token` | 生成订阅链接（90 天，旧链接失效）：`{token, url, webcal, expireAt}` |
| `GET /schedule/ical/token-info` | 当前订阅链接 `{token, expireAt}`，没有为 `null` |
| `GET /schedule/ical/subscribe?token=`（公开） | 日历应用订阅地址，返回课表快照生成的 iCalendar |

课程对象：`{name, teacher, room, dayOfWeek(1-7), periodStart, periods, weeks?}`

## 成绩

| 接口 | 说明 |
|---|---|
| `GET /score?semester=2025-2026-1` | 成绩列表（缓存 15 分钟），不传 `semester` 返回全部 |
| `GET /score/semesters` | 有成绩的学期，最新在前 |
| `GET /score/stats` | `{totalCredits, weightedGPA, simpleGPA, failedCount, totalCourses, semesterStats[], highestCourse, lowestCourse, failedCourses[]}` |

## 通知与监控

| 接口 | 说明 |
|---|---|
| `GET /monitor/status` | `{monitoring, notifyChannel, lastCheck, lastCheckError, lastKeepalive, lastKeepaliveError, keepaliveMinutes, checkMinutes, history[{event,text,time}]}` |
| `GET /notify/channels` | `{channels: {pushplus, email, bot}, emailAvailable}` |
| `PUT /notify/channels` | 整体覆盖，未提供的渠道视为关闭：`{pushplus?: {token}, email?: {to}, bot?: {type: wecom\|dingtalk\|feishu, url, secret?}}` |
| `POST /notify/test` | 向所有已开启的渠道（含 Webhook）发送测试消息：`{results: [{channel, error?}]}` |
| `GET /notify/reminder` / `PUT /notify/reminder` | 上课提醒：`{daily, dailyTime: "07:00", dailyTomorrow, beforeClass, beforeMinutes}` |
| `POST /webhook/register` | `{url, secret?}` |
| `GET /webhook/info` | `{registered, url, secret(打码), registeredAt}` |
| `POST /webhook/trigger` | 把最近一次课表变动重新推送到 Webhook |

### Webhook 请求

`POST application/json`，请求头 `X-JWW-Event` 为事件类型；配置了密钥时带 `X-Hub-Signature-256: sha256=<HMAC-SHA256(body, secret)>`。

```json
{
  "event": "schedule-diff",
  "text": "课表有变动：\n【加课】\n+ 大学物理 周三 第1-2节 @B202（第4周）",
  "data": { "added": [], "removed": [], "changed": [], "detectedAt": "..." },
  "time": "2026-10-09T08:00:00+08:00"
}
```

| event | 含义 | data |
|---|---|---|
| `schedule-diff` | 课表变动（只比较当前周及以后） | 同 `/schedule/diff` |
| `score-new` | 新出或被修改的成绩 | `[{score, oldGrade?}]` |
| `reminder` | 上课提醒 | 无 |
| `session-expired` | 教务登录失效，需要重新登录 | 无 |
| `test` | 测试 | 无 |

## 健康检查

`GET /health` → `{"ok": 1}`
