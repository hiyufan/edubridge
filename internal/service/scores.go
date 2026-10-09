package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/redis/go-redis/v9"
	"jww/internal/model"
	"jww/pkg/database"
)

// 成绩发布提醒：记住用户已有的成绩，发现新出的成绩（或成绩被修改）时推送。
// 第一次完整拉取只建立基线；基线完整之前（部分分页失败）不通知，避免把老成绩当成新成绩。

const knownScoresPrefix = "score:known:" // uid -> knownScores JSON

type knownScores struct {
	Complete bool              `json:"complete"`
	Grades   map[string]string `json:"grades"` // 学年|学期|课程 -> 成绩
}

// ScoreChange 新出或被修改的成绩
type ScoreChange struct {
	Score    model.Score `json:"score"`
	OldGrade string      `json:"oldGrade,omitempty"` // 非空表示成绩被修改
}

func scoreKey(sc model.Score) string {
	return sc.Year + "|" + sc.Semester + "|" + sc.Course
}

func gradeString(g interface{}) string {
	if g == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(g))
}

func loadKnownScores(uid string) (*knownScores, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	data, err := database.GetRedis().Get(ctx, knownScoresPrefix+uid).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var k knownScores
	if err := json.Unmarshal(data, &k); err != nil {
		return nil, err
	}
	if k.Grades == nil {
		k.Grades = map[string]string{}
	}
	return &k, nil
}

func saveKnownScores(uid string, k *knownScores) error {
	data, err := json.Marshal(k)
	if err != nil {
		return err
	}
	ctx, cancel := storeCtx()
	defer cancel()
	return database.GetRedis().Set(ctx, knownScoresPrefix+uid, data, 0).Err()
}

// diffScores 与已知成绩比对，返回新出/被修改的成绩，并把本次结果合并进 known
func diffScores(known *knownScores, scores []model.Score) []ScoreChange {
	var changes []ScoreChange
	for _, sc := range scores {
		key := scoreKey(sc)
		grade := gradeString(sc.Grade)
		if grade == "" {
			continue // 尚未录入成绩
		}
		old, ok := known.Grades[key]
		switch {
		case !ok:
			changes = append(changes, ScoreChange{Score: sc})
		case old != grade:
			changes = append(changes, ScoreChange{Score: sc, OldGrade: old})
		}
		known.Grades[key] = grade
	}
	sort.Slice(changes, func(i, j int) bool {
		return scoreKey(changes[i].Score) < scoreKey(changes[j].Score)
	})
	return changes
}

// recordScores 记录一次成绩拉取结果，有新成绩时通知
func (s *JwService) recordScores(uid string, scores []model.Score, complete bool) {
	s.snapMu.Lock()
	defer s.snapMu.Unlock()

	known, err := loadKnownScores(uid)
	if errors.Is(err, ErrNotFound) {
		known = &knownScores{Grades: map[string]string{}}
	} else if err != nil {
		slog.Warn("Load known scores failed", "uid", uid, "err", err)
		return
	}

	baselineReady := known.Complete
	changes := diffScores(known, scores)
	if complete {
		known.Complete = true
	}
	if err := saveKnownScores(uid, known); err != nil {
		slog.Warn("Save known scores failed", "uid", uid, "err", err)
		return
	}

	if baselineReady && len(changes) > 0 {
		slog.Info("New scores detected", "uid", uid, "count", len(changes))
		notifyUser(uid, EventScoreNew, scoreSummary(changes), changes)
	}
}

func formatNumber(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", f), "0"), ".")
}

func scoreSummary(changes []ScoreChange) string {
	var b strings.Builder
	b.WriteString("出成绩了：")
	for _, ch := range changes {
		sc := ch.Score
		line := fmt.Sprintf("\n《%s》%s", sc.Course, gradeString(sc.Grade))
		var extra []string
		if sc.GPA > 0 {
			extra = append(extra, "绩点 "+formatNumber(sc.GPA))
		}
		if sc.Credit > 0 {
			extra = append(extra, "学分 "+formatNumber(sc.Credit))
		}
		if len(extra) > 0 {
			line += "（" + strings.Join(extra, "，") + "）"
		}
		if ch.OldGrade != "" {
			line += fmt.Sprintf(" [由 %s 修改]", ch.OldGrade)
		}
		b.WriteString(line)
	}
	return b.String()
}

// checkScores 后台监控中拉取一次成绩
func (s *JwService) checkScores(session *Session) error {
	scores, complete, err := s.fetchAllScores(session)
	if err != nil {
		if errors.Is(err, ErrSessionExpired) {
			s.expireUser(session.UID)
		}
		return err
	}
	s.persistCookies(session)
	s.recordScores(session.UID, scores, complete)
	return nil
}
