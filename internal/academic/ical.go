package academic

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"jww/internal/model"
	"jww/internal/timetable"
)

// ICalTokenTTL 订阅链接有效期
const ICalTokenTTL = 90 * 24 * time.Hour

// CreateICalToken 生成新的订阅 token（旧的立即失效）。先拉一次课表，确保订阅时有课表快照。
func (s *Service) CreateICalToken(ctx context.Context, uid string) (token string, expireAt time.Time, err error) {
	if _, err := s.FullSchedule(ctx, uid, MaxWeek); err != nil {
		return "", time.Time{}, err
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", time.Time{}, err
	}
	token = base64.URLEncoding.EncodeToString(b)
	if err := s.store.SaveICalToken(ctx, uid, token, ICalTokenTTL); err != nil {
		return "", time.Time{}, err
	}
	return token, s.now().Add(ICalTokenTTL), nil
}

// ICalToken 用户当前的订阅 token
func (s *Service) ICalToken(ctx context.Context, uid string) (string, time.Time, error) {
	return s.store.UserICalToken(ctx, uid)
}

// ICalByToken 免登录订阅：用 token 找到用户，读取其最近一次课表快照生成 iCal
// （不依赖教务登录态，课表由后台监控定时刷新）
func (s *Service) ICalByToken(ctx context.Context, token string) (ics string, schedule *model.FullSchedule, err error) {
	uid, err := s.store.ICalTokenUser(ctx, token)
	if err != nil {
		return "", nil, err
	}
	schedule, _, err = s.store.Snapshot(ctx, uid)
	if err != nil {
		return "", nil, err
	}
	return timetable.ICal(schedule, s.now()), schedule, nil
}

// ICal 当前用户的 iCal（实时拉取）
func (s *Service) ICal(ctx context.Context, uid string) (string, *model.FullSchedule, error) {
	full, err := s.FullSchedule(ctx, uid, MaxWeek)
	if err != nil {
		return "", nil, err
	}
	return timetable.ICal(full, s.now()), full, nil
}
