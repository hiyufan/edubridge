package store

// Redis key 一览（与旧版本保持一致，升级无需迁移数据）
const (
	keyCookies        = "jw:cookies:"         // uid -> 教务 cookie JSON
	keyMonitoredUsers = "jw:monitor:users"    // set: 处于监控中的学号
	keySnapshot       = "schedule:snapshot:"  // uid -> 最近一次课表 {schedule, complete}
	keyLatestDiff     = "schedule:diff:"      // uid -> 最近一次课表变动
	keyKnownScores    = "score:known:"        // uid -> 已知成绩
	keyWebhook        = "webhook:"            // uid -> Webhook 配置
	keyChannels       = "notify:channels:"    // uid -> 通知渠道
	keyHistory        = "notify:history:"     // uid -> list: 最近动态（新的在前）
	keyMonitorStatus  = "monitor:status:"     // uid -> hash: 上次检查/保活
	keyReminder       = "reminder:settings:"  // uid -> 上课提醒设置
	keyReminderUsers  = "reminder:users"      // set: 开启了上课提醒的学号
	keyReminderSent   = "reminder:sent:"      // uid:提醒标识 -> 已发送标记
	keyICalToken      = "ical:token:"         // token -> uid
	keyICalUser       = "ical:user:"          // uid -> token
	keyRefreshToken   = "refresh_token:"      // tokenID -> hash{user_id, expires_at}
	keyRefreshByUser  = "refresh_token:user:" // uid -> set: tokenID
)
