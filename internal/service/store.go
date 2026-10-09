package service

import (
	"context"
	"encoding/json"
	"errors"
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

// saveScheduleSnapshot 保存用户最近一次完整课表，供 iCal 订阅在无登录会话时使用
func saveScheduleSnapshot(uid string, schedule *model.FullSchedule) error {
	data, err := json.Marshal(schedule)
	if err != nil {
		return err
	}
	ctx, cancel := storeCtx()
	defer cancel()
	return database.GetRedis().Set(ctx, scheduleSnapPrefix+uid, data, scheduleSnapshotTTL).Err()
}

// GetScheduleSnapshot 读取用户最近一次完整课表
func GetScheduleSnapshot(uid string) (*model.FullSchedule, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	data, err := database.GetRedis().Get(ctx, scheduleSnapPrefix+uid).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var schedule model.FullSchedule
	if err := json.Unmarshal(data, &schedule); err != nil {
		return nil, err
	}
	return &schedule, nil
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
