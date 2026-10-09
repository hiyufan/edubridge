package store

import (
	"context"
	"errors"
	"time"

	"jww/internal/model"
)

const (
	snapshotTTL   = 180 * 24 * time.Hour
	latestDiffTTL = 30 * 24 * time.Hour
)

type snapshotRecord struct {
	Schedule *model.FullSchedule `json:"schedule"`
	Complete bool                `json:"complete"` // 所有周都成功拉取，只有完整快照才参与比对
}

// SaveSnapshot 保存用户最近一次课表
func (s *Store) SaveSnapshot(ctx context.Context, uid string, schedule *model.FullSchedule, complete bool) error {
	return s.setJSON(ctx, keySnapshot+uid, snapshotRecord{Schedule: schedule, Complete: complete}, snapshotTTL)
}

// Snapshot 读取用户最近一次课表
func (s *Store) Snapshot(ctx context.Context, uid string) (schedule *model.FullSchedule, complete bool, err error) {
	var rec snapshotRecord
	if err := s.getJSON(ctx, keySnapshot+uid, &rec); err != nil {
		return nil, false, err
	}
	if rec.Schedule == nil {
		return nil, false, ErrNotFound
	}
	return rec.Schedule, rec.Complete, nil
}

// SaveLatestDiff 保存最近一次课表变动
func (s *Store) SaveLatestDiff(ctx context.Context, uid string, d *model.ScheduleDiff) error {
	return s.setJSON(ctx, keyLatestDiff+uid, d, latestDiffTTL)
}

// LatestDiff 最近一次课表变动
func (s *Store) LatestDiff(ctx context.Context, uid string) (*model.ScheduleDiff, error) {
	var d model.ScheduleDiff
	if err := s.getJSON(ctx, keyLatestDiff+uid, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// KnownScores 已知成绩；没有记录时返回空记录
func (s *Store) KnownScores(ctx context.Context, uid string) (*model.KnownScores, error) {
	k := &model.KnownScores{}
	if err := s.getJSON(ctx, keyKnownScores+uid, k); err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if k.Grades == nil {
		k.Grades = map[string]string{}
	}
	return k, nil
}

// SaveKnownScores 保存已知成绩
func (s *Store) SaveKnownScores(ctx context.Context, uid string, k *model.KnownScores) error {
	return s.setJSON(ctx, keyKnownScores+uid, k, 0)
}
