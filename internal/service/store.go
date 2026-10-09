package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"jww/internal/model"
	"jww/pkg/database"
)

// 持久化存储（Redis），按学号（UID）而非教务会话 ID 关联，
// 这样服务重启或教务会话 30 分钟过期后 iCal 订阅和 Webhook 依然有效。
const (
	iCalTokenPrefix     = "ical:token:"        // token -> uid
	iCalUserPrefix      = "ical:user:"         // uid -> token
	scheduleSnapPrefix  = "schedule:snapshot:" // uid -> FullSchedule JSON
	webhookPrefix       = "webhook:"           // uid -> WebhookEntry JSON
	ICalTokenTTL        = 90 * 24 * time.Hour
	scheduleSnapshotTTL = 180 * 24 * time.Hour
)

// ErrNotFound 存储中不存在或已过期
var ErrNotFound = errors.New("not found")

// WebhookEntry webhook 注册项
type WebhookEntry struct {
	URL        string    `json:"url"`
	Secret     string    `json:"secret"`
	Registered time.Time `json:"registered"`
}

func storeCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 3*time.Second)
}

// SaveICalToken 保存 iCal token，同一用户只保留最新一个
func SaveICalToken(uid, token string) error {
	ctx, cancel := storeCtx()
	defer cancel()
	rdb := database.GetRedis()

	if old, err := rdb.Get(ctx, iCalUserPrefix+uid).Result(); err == nil {
		rdb.Del(ctx, iCalTokenPrefix+old)
	}

	_, err := rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Set(ctx, iCalTokenPrefix+token, uid, ICalTokenTTL)
		p.Set(ctx, iCalUserPrefix+uid, token, ICalTokenTTL)
		return nil
	})
	return err
}

// LookupICalToken 由 token 查学号
func LookupICalToken(token string) (string, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	uid, err := database.GetRedis().Get(ctx, iCalTokenPrefix+token).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	return uid, err
}

// GetUserICalToken 查用户当前的 iCal token 及过期时间
func GetUserICalToken(uid string) (string, time.Time, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	rdb := database.GetRedis()

	token, err := rdb.Get(ctx, iCalUserPrefix+uid).Result()
	if errors.Is(err, redis.Nil) {
		return "", time.Time{}, ErrNotFound
	}
	if err != nil {
		return "", time.Time{}, err
	}
	ttl, err := rdb.TTL(ctx, iCalUserPrefix+uid).Result()
	if err != nil {
		return "", time.Time{}, err
	}
	return token, time.Now().Add(ttl), nil
}

// snapshotRecord 课表快照；Complete 表示所有周都成功拉取，只有完整快照才参与变动比对
type snapshotRecord struct {
	Schedule *model.FullSchedule `json:"schedule"`
	Complete bool                `json:"complete"`
}

// saveScheduleSnapshot 保存用户最近一次课表，供变动比对和 iCal 订阅（无登录会话时）使用
func saveScheduleSnapshot(uid string, schedule *model.FullSchedule, complete bool) error {
	data, err := json.Marshal(&snapshotRecord{Schedule: schedule, Complete: complete})
	if err != nil {
		return err
	}
	ctx, cancel := storeCtx()
	defer cancel()
	return database.GetRedis().Set(ctx, scheduleSnapPrefix+uid, data, scheduleSnapshotTTL).Err()
}

func getSnapshotRecord(uid string) (*model.FullSchedule, bool, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	data, err := database.GetRedis().Get(ctx, scheduleSnapPrefix+uid).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, ErrNotFound
	}
	if err != nil {
		return nil, false, err
	}
	var rec snapshotRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, false, err
	}
	if rec.Schedule == nil { // 旧格式快照，视为不存在
		return nil, false, ErrNotFound
	}
	return rec.Schedule, rec.Complete, nil
}

// GetScheduleSnapshot 读取用户最近一次课表
func GetScheduleSnapshot(uid string) (*model.FullSchedule, error) {
	schedule, _, err := getSnapshotRecord(uid)
	return schedule, err
}

// SaveWebhook 注册（覆盖）用户的 webhook
func SaveWebhook(uid string, entry *WebhookEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	ctx, cancel := storeCtx()
	defer cancel()
	return database.GetRedis().Set(ctx, webhookPrefix+uid, data, 0).Err()
}

// GetWebhook 获取用户的 webhook
func GetWebhook(uid string) (*WebhookEntry, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	data, err := database.GetRedis().Get(ctx, webhookPrefix+uid).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var entry WebhookEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// ---- 教务登录态 ----

const (
	jwCookiesPrefix = "jw:cookies:" // uid -> cookies JSON
	jwSIDPrefix     = "jw:sid:"     // 前端 sessionID -> uid
	jwMonitorUsers  = "jw:monitor:users"
	scheduleDiffKey = "schedule:diff:"    // uid -> 最近一次 ScheduleDiff JSON
	jwSIDTTL        = 30 * 24 * time.Hour // 与 refresh token 有效期一致
	scheduleDiffTTL = 30 * 24 * time.Hour
)

func saveJwCookies(uid string, cookies map[string][]*http.Cookie) error {
	data, err := json.Marshal(cookies)
	if err != nil {
		return err
	}
	ctx, cancel := storeCtx()
	defer cancel()
	return database.GetRedis().Set(ctx, jwCookiesPrefix+uid, data, 0).Err()
}

func loadJwCookies(uid string) (map[string][]*http.Cookie, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	data, err := database.GetRedis().Get(ctx, jwCookiesPrefix+uid).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var cookies map[string][]*http.Cookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return nil, err
	}
	return cookies, nil
}

func deleteJwCookies(uid string) error {
	ctx, cancel := storeCtx()
	defer cancel()
	return database.GetRedis().Del(ctx, jwCookiesPrefix+uid).Err()
}

func bindSessionID(sessionID, uid string) error {
	ctx, cancel := storeCtx()
	defer cancel()
	return database.GetRedis().Set(ctx, jwSIDPrefix+sessionID, uid, jwSIDTTL).Err()
}

func lookupSessionID(sessionID string) (string, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	rdb := database.GetRedis()
	uid, err := rdb.Get(ctx, jwSIDPrefix+sessionID).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	rdb.Expire(ctx, jwSIDPrefix+sessionID, jwSIDTTL) // 使用即续期
	return uid, nil
}

func addMonitorUser(uid string) error {
	ctx, cancel := storeCtx()
	defer cancel()
	return database.GetRedis().SAdd(ctx, jwMonitorUsers, uid).Err()
}

// removeMonitorUser 返回该用户此前是否在监控列表中
func removeMonitorUser(uid string) (bool, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	n, err := database.GetRedis().SRem(ctx, jwMonitorUsers, uid).Result()
	return n > 0, err
}

func monitorUsers() ([]string, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	return database.GetRedis().SMembers(ctx, jwMonitorUsers).Result()
}

// IsMonitored 用户是否处于课表监控中
func IsMonitored(uid string) (bool, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	return database.GetRedis().SIsMember(ctx, jwMonitorUsers, uid).Result()
}

func saveLatestDiff(uid string, diff *ScheduleDiff) error {
	data, err := json.Marshal(diff)
	if err != nil {
		return err
	}
	ctx, cancel := storeCtx()
	defer cancel()
	return database.GetRedis().Set(ctx, scheduleDiffKey+uid, data, scheduleDiffTTL).Err()
}

// GetLatestDiff 获取用户最近一次检测到的课表变动
func GetLatestDiff(uid string) (*ScheduleDiff, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	data, err := database.GetRedis().Get(ctx, scheduleDiffKey+uid).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var diff ScheduleDiff
	if err := json.Unmarshal(data, &diff); err != nil {
		return nil, err
	}
	return &diff, nil
}

// ---- 监控状态与动态历史 ----

const (
	monitorStatusPrefix = "monitor:status:" // uid -> hash
	historyPrefix       = "notify:history:" // uid -> list of HistoryEntry JSON（新的在前）
	historyMax          = 30
)

// MonitorStatus 用户的后台监控运行情况
type MonitorStatus struct {
	LastCheck          *time.Time `json:"lastCheck"`
	LastCheckError     string     `json:"lastCheckError,omitempty"`
	LastKeepalive      *time.Time `json:"lastKeepalive"`
	LastKeepaliveError string     `json:"lastKeepaliveError,omitempty"`
}

func recordMonitorResult(uid, kind string, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	ctx, cancel := storeCtx()
	defer cancel()
	if e := database.GetRedis().HSet(ctx, monitorStatusPrefix+uid,
		kind, time.Now().Unix(), kind+"Error", msg).Err(); e != nil {
		slog.Warn("Record monitor status failed", "uid", uid, "err", e)
	}
}

// GetMonitorStatus 读取监控运行情况
func GetMonitorStatus(uid string) (*MonitorStatus, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	m, err := database.GetRedis().HGetAll(ctx, monitorStatusPrefix+uid).Result()
	if err != nil {
		return nil, err
	}
	parse := func(k string) *time.Time {
		sec, err := strconv.ParseInt(m[k], 10, 64)
		if err != nil {
			return nil
		}
		t := time.Unix(sec, 0)
		return &t
	}
	return &MonitorStatus{
		LastCheck:          parse("lastCheck"),
		LastCheckError:     m["lastCheckError"],
		LastKeepalive:      parse("lastKeepalive"),
		LastKeepaliveError: m["lastKeepaliveError"],
	}, nil
}

// HistoryEntry 一条动态（课表变动、新成绩、登录失效）
type HistoryEntry struct {
	Event string    `json:"event"`
	Text  string    `json:"text"`
	Time  time.Time `json:"time"`
}

func pushHistory(uid string, e *HistoryEntry) {
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	ctx, cancel := storeCtx()
	defer cancel()
	_, err = database.GetRedis().TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.LPush(ctx, historyPrefix+uid, data)
		p.LTrim(ctx, historyPrefix+uid, 0, historyMax-1)
		return nil
	})
	if err != nil {
		slog.Warn("Push history failed", "uid", uid, "err", err)
	}
}

// GetHistory 最近的动态（新的在前）
func GetHistory(uid string, limit int) ([]HistoryEntry, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	items, err := database.GetRedis().LRange(ctx, historyPrefix+uid, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]HistoryEntry, 0, len(items))
	for _, it := range items {
		var e HistoryEntry
		if json.Unmarshal([]byte(it), &e) == nil {
			out = append(out, e)
		}
	}
	return out, nil
}
