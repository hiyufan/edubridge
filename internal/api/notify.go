package api

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"jww/internal/model"
	"jww/internal/monitor"
	"jww/internal/notify"
	"jww/internal/store"
	"jww/internal/timetable"
)

func (a *API) registerWebhook(c *gin.Context) {
	var req struct {
		URL    string `json:"url" binding:"required,url"`
		Secret string `json:"secret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请填写正确的 Webhook URL")
		return
	}
	if err := a.Notifier.SaveWebhook(c.Request.Context(), uidOf(c), req.URL, req.Secret); err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"message": "webhook 注册成功"})
}

func (a *API) webhookInfo(c *gin.Context) {
	w, err := a.Notifier.Webhook(c.Request.Context(), uidOf(c))
	if errors.Is(err, store.ErrNotFound) {
		ok(c, gin.H{"registered": false})
		return
	}
	if err != nil {
		fail(c, err)
		return
	}
	masked := ""
	if len(w.Secret) > 8 {
		masked = w.Secret[:8] + "***"
	}
	ok(c, gin.H{
		"registered":   true,
		"url":          w.URL,
		"secret":       masked,
		"registeredAt": w.Registered.In(timetable.CST).Format("2006-01-02 15:04"),
	})
}

// triggerWebhook 把最近一次课表变动重新推送到 Webhook
func (a *API) triggerWebhook(c *gin.Context) {
	ctx := c.Request.Context()
	w, err := a.Notifier.Webhook(ctx, uidOf(c))
	if errors.Is(err, store.ErrNotFound) {
		badRequest(c, "未注册 webhook")
		return
	}
	if err != nil {
		fail(c, err)
		return
	}
	d, err := a.Academic.LatestDiff(ctx, uidOf(c))
	if err != nil {
		fail(c, err)
		return
	}
	summary := timetable.DiffSummary(d)
	p := &notify.Payload{Event: notify.EventScheduleDiff, Text: summary, Data: d, Time: time.Now()}
	if err := a.Notifier.SendWebhook(ctx, w, p); err != nil {
		badRequest(c, "推送失败: "+err.Error())
		return
	}
	ok(c, gin.H{"message": "推送成功", "summary": summary})
}

func (a *API) channels(c *gin.Context) {
	ch, err := a.Notifier.Channels(c.Request.Context(), uidOf(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"channels": ch, "emailAvailable": a.Notifier.EmailAvailable()})
}

// saveChannels 整体覆盖通知渠道，未提供的渠道视为关闭
func (a *API) saveChannels(c *gin.Context) {
	var ch model.NotifyChannels
	if err := c.ShouldBindJSON(&ch); err != nil {
		badRequest(c, "参数格式不正确")
		return
	}
	if err := a.Notifier.SaveChannels(c.Request.Context(), uidOf(c), &ch); err != nil {
		fail(c, err) // 校验失败为 400
		return
	}
	ok(c, gin.H{"channels": ch})
}

// testNotify 向所有已开启的渠道发送测试消息，按渠道返回结果（部分失败也返回 200）
func (a *API) testNotify(c *gin.Context) {
	results, err := a.Notifier.Send(c.Request.Context(), uidOf(c), &notify.Payload{
		Event: notify.EventTest,
		Text:  "这是一条测试通知，课表有加课、减课时会通过这里提醒你。",
		Time:  time.Now(),
	})
	if err != nil && results == nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"results": results})
}

func (a *API) reminder(c *gin.Context) {
	r, err := a.Reminder.Settings(c.Request.Context(), uidOf(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, r)
}

func (a *API) saveReminder(c *gin.Context) {
	var r model.ReminderSettings
	if err := c.ShouldBindJSON(&r); err != nil {
		badRequest(c, "参数格式不正确")
		return
	}
	if err := a.Reminder.Save(c.Request.Context(), uidOf(c), &r); err != nil {
		fail(c, err) // 校验失败为 400
		return
	}
	ok(c, r)
}

func (a *API) monitorStatus(c *gin.Context) {
	ctx, uid := c.Request.Context(), uidOf(c)
	status, err := a.Monitor.Status(ctx, uid)
	if err != nil {
		fail(c, err)
		return
	}
	history, err := a.Notifier.History(ctx, uid, 20)
	if err != nil {
		fail(c, err)
		return
	}
	hasChannel, err := a.Notifier.HasChannel(ctx, uid)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, struct {
		*monitor.Status
		NotifyChannel bool                 `json:"notifyChannel"`
		History       []model.HistoryEntry `json:"history"`
	}{status, hasChannel, history})
}
