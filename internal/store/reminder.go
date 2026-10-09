package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"jww/internal/model"
)

const reminderSentTTL = 48 * time.Hour

// SaveReminder 保存提醒设置，并维护开启提醒的用户集合
func (s *Store) SaveReminder(ctx context.Context, uid string, r *model.ReminderSettings) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = s.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Set(ctx, keyReminder+uid, data, 0)
		if r.Daily || r.BeforeClass {
			p.SAdd(ctx, keyReminderUsers, uid)
		} else {
			p.SRem(ctx, keyReminderUsers, uid)
		}
		return nil
	})
	return err
}

// Reminder 提醒设置；未设置时返回 ErrNotFound
func (s *Store) Reminder(ctx context.Context, uid string) (*model.ReminderSettings, error) {
	var r model.ReminderSettings
	if err := s.getJSON(ctx, keyReminder+uid, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ReminderUsers 开启了上课提醒的学号
func (s *Store) ReminderUsers(ctx context.Context) ([]string, error) {
	return s.rdb.SMembers(ctx, keyReminderUsers).Result()
}

// MarkReminderSent 标记某条提醒已发送；之前已标记过返回 false
func (s *Store) MarkReminderSent(ctx context.Context, uid, id string) (bool, error) {
	return s.rdb.SetNX(ctx, keyReminderSent+uid+":"+id, 1, reminderSentTTL).Result()
}
