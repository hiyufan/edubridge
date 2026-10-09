package academic

import (
	"context"
	"log/slog"

	"jww/internal/grades"
	"jww/internal/model"
	"jww/internal/notify"
)

type scoresEntry struct {
	scores []model.Score
}

// Scores 成绩（缓存 15 分钟）；semester 形如 "2025-2026-1"，为空表示全部
func (s *Service) Scores(ctx context.Context, uid, semester string) ([]model.Score, error) {
	all, err := s.allScores(ctx, uid)
	if err != nil {
		return nil, err
	}
	return grades.Filter(all, semester), nil
}

// Semesters 有成绩的学期，最新的在前
func (s *Service) Semesters(ctx context.Context, uid string) ([]string, error) {
	all, err := s.allScores(ctx, uid)
	if err != nil {
		return nil, err
	}
	return grades.Semesters(all), nil
}

// ScoreStats 成绩统计
func (s *Service) ScoreStats(ctx context.Context, uid string) (model.ScoreStats, error) {
	all, err := s.allScores(ctx, uid)
	if err != nil {
		return model.ScoreStats{}, err
	}
	return grades.Stats(all), nil
}

// RefreshScores 跳过缓存重新拉取成绩并检测新成绩（后台监控用）
func (s *Service) RefreshScores(ctx context.Context, uid string) error {
	_, err := s.fetchScores(ctx, uid)
	return err
}

func (s *Service) allScores(ctx context.Context, uid string) ([]model.Score, error) {
	if e, ok := s.scores.get(uid, s.now()); ok {
		return e.scores, nil
	}
	return s.fetchScores(ctx, uid)
}

func (s *Service) fetchScores(ctx context.Context, uid string) ([]model.Score, error) {
	c, err := s.client(ctx, uid)
	if err != nil {
		return nil, err
	}
	all, complete, err := c.Scores(ctx)
	if err := s.done(ctx, uid, c, err); err != nil {
		return nil, err
	}
	s.scores.set(uid, scoresEntry{all}, s.now().Add(scoreCacheTTL))
	s.recordScores(ctx, uid, all, complete)
	return all, nil
}

// recordScores 把本次成绩并入已知成绩；基线完整后发现的新成绩才通知
func (s *Service) recordScores(ctx context.Context, uid string, all []model.Score, complete bool) {
	s.recordMu.Lock()
	defer s.recordMu.Unlock()

	known, err := s.store.KnownScores(ctx, uid)
	if err != nil {
		slog.Warn("load known scores failed", "uid", uid, "err", err)
		return
	}
	changes, ready := grades.Track(known, all, complete)
	if err := s.store.SaveKnownScores(ctx, uid, known); err != nil {
		slog.Warn("save known scores failed", "uid", uid, "err", err)
		return
	}
	if ready && len(changes) > 0 {
		slog.Info("new scores", "uid", uid, "count", len(changes))
		s.notifier.Notify(uid, notify.EventScoreNew, grades.ChangesSummary(changes), changes)
	}
}
