package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
	_ "time/tzdata" // 容器镜像（alpine）可能没有时区数据

	"github.com/redis/go-redis/v9"
	"jww/internal/model"
	"jww/pkg/database"
)

// 上课提醒：基于 Redis 中的课表快照（后台监控每小时刷新），按北京时间推送。
//   - 每日课表：每天在设定时间推送今天（或明天）的课；
//   - 课前提醒：每节课开始前 N 分钟推送。

var cst = loadCST()

func loadCST() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

// periodTimes 节次作息（与前端 utils/periods.js 保持一致）
var periodTimes = map[int][2]string{
	1: {"08:00", "08:45"}, 2: {"08:55", "09:40"}, 3: {"10:00", "10:45"}, 4: {"10:55", "11:40"},
	5: {"14:00", "14:45"}, 6: {"14:55", "15:40"}, 7: {"15:50", "16:35"}, 8: {"16:45", "17:30"},
	9: {"18:30", "19:15"}, 10: {"19:25", "20:10"}, 11: {"20:20", "21:05"}, 12: {"21:15", "22:00"},
}

// PeriodTimeRange 返回连续节次的起止时间（"08:00"）；未知节次返回空字符串
func PeriodTimeRange(periodStart, periods int) (start, end string) {
	if periods < 1 {
		periods = 1
	}
	start = periodTimes[periodStart][0]
	end = periodTimes[periodStart+periods-1][1]
	if end == "" {
		end = periodTimes[periodStart][1]
	}
	return start, end
}

// ReminderSettings 用户的上课提醒设置
type ReminderSettings struct {
	Daily         bool   `json:"daily"`
	DailyTime     string `json:"dailyTime"`     // "07:00"
	DailyTomorrow bool   `json:"dailyTomorrow"` // true 时推送明天的课（适合设在晚上）
	BeforeClass   bool   `json:"beforeClass"`
	BeforeMinutes int    `json:"beforeMinutes"` // 课前多少分钟
}

// DefaultReminderSettings 未设置时的默认值（均关闭）
func DefaultReminderSettings() *ReminderSettings {
	return &ReminderSettings{DailyTime: "07:00", BeforeMinutes: 15}
}

// Validate 校验设置
func (r *ReminderSettings) Validate() error {
	if r.DailyTime == "" {
		r.DailyTime = "07:00"
	}
	if _, err := time.Parse("15:04", r.DailyTime); err != nil {
		return errors.New("提醒时间格式应为 HH:MM")
	}
	if r.BeforeMinutes == 0 {
		r.BeforeMinutes = 15
	}
	if r.BeforeMinutes < 1 || r.BeforeMinutes > 180 {
		return errors.New("课前提醒时间应在 1 到 180 分钟之间")
	}
	return nil
}

const (
	reminderSettingsPrefix = "reminder:settings:"
	reminderUsers          = "reminder:users"
	reminderSentPrefix     = "reminder:sent:"
	reminderSentTTL        = 48 * time.Hour
)

// SaveReminderSettings 保存提醒设置，并维护需要提醒的用户集合
func SaveReminderSettings(uid string, r *ReminderSettings) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	ctx, cancel := storeCtx()
	defer cancel()
	_, err = database.GetRedis().TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Set(ctx, reminderSettingsPrefix+uid, data, 0)
		if r.Daily || r.BeforeClass {
			p.SAdd(ctx, reminderUsers, uid)
		} else {
			p.SRem(ctx, reminderUsers, uid)
		}
		return nil
	})
	return err
}

// GetReminderSettings 读取提醒设置，未设置时返回默认值
func GetReminderSettings(uid string) (*ReminderSettings, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	data, err := database.GetRedis().Get(ctx, reminderSettingsPrefix+uid).Bytes()
	if errors.Is(err, redis.Nil) {
		return DefaultReminderSettings(), nil
	}
	if err != nil {
		return nil, err
	}
	var r ReminderSettings
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// markSent 标记某条提醒已发送；已发送过返回 false
func markSent(uid, key string) bool {
	ctx, cancel := storeCtx()
	defer cancel()
	ok, err := database.GetRedis().SetNX(ctx, reminderSentPrefix+uid+":"+key, 1, reminderSentTTL).Result()
	if err != nil {
		slog.Warn("Mark reminder sent failed", "uid", uid, "err", err)
		return false
	}
	return ok
}

// weekOf 计算某天是第几教学周（学期开始前返回 0）
func weekOf(semesterStart string, day time.Time) int {
	start, err := time.ParseInLocation("2006-01-02", semesterStart, cst)
	if err != nil {
		return 0
	}
	d := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, cst)
	days := int(d.Sub(start).Hours()/24 + 0.5)
	if days < 0 {
		return 0
	}
	return days/7 + 1
}

func weekdayOf(day time.Time) int {
	wd := int(day.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}

// classesOn 某天的课，按节次排序
func classesOn(schedule *model.FullSchedule, day time.Time) []model.Course {
	week := weekOf(schedule.SemesterStart, day)
	if week == 0 {
		return nil
	}
	wd := weekdayOf(day)
	var out []model.Course
	for _, c := range schedule.Courses {
		if c.DayOfWeek != wd {
			continue
		}
		for _, w := range c.Weeks {
			if w == week {
				out = append(out, c)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PeriodStart < out[j].PeriodStart })
	return out
}

func describeClass(c model.Course) string {
	start, end := PeriodTimeRange(c.PeriodStart, c.Periods)
	s := c.Name
	if start != "" {
		s = fmt.Sprintf("%s-%s %s", start, end, c.Name)
	}
	if c.Room != "" {
		s += " @" + c.Room
	}
	return s
}

func dailyText(courses []model.Course, day time.Time, tomorrow bool) string {
	label := "今天"
	if tomorrow {
		label = "明天"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s（%s）有 %d 节课：", label, weekdayNames[weekdayOf(day)], len(courses))
	for _, c := range courses {
		b.WriteString("\n" + describeClass(c))
	}
	return b.String()
}

// StartReminders 启动上课提醒（随 Close 停止）
func (s *JwService) StartReminders() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.runReminders(time.Now().In(cst))
			}
		}
	}()
}

// runReminders 检查所有开启提醒的用户，发送到点的提醒
func (s *JwService) runReminders(now time.Time) {
	ctx, cancel := storeCtx()
	uids, err := database.GetRedis().SMembers(ctx, reminderUsers).Result()
	cancel()
	if err != nil {
		slog.Warn("Load reminder users failed", "err", err)
		return
	}
	for _, uid := range uids {
		s.remindUser(uid, now)
	}
}

func (s *JwService) remindUser(uid string, now time.Time) {
	settings, err := GetReminderSettings(uid)
	if err != nil {
		slog.Warn("Load reminder settings failed", "uid", uid, "err", err)
		return
	}
	schedule, err := GetScheduleSnapshot(uid)
	if err != nil {
		return // 还没有课表
	}
	today := now.Format("2006-01-02")

	if settings.Daily {
		t, _ := time.Parse("15:04", settings.DailyTime)
		target := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, cst)
		if !now.Before(target) && now.Sub(target) < 10*time.Minute && markSent(uid, "daily:"+today) {
			day := now
			if settings.DailyTomorrow {
				day = now.AddDate(0, 0, 1)
			}
			if courses := classesOn(schedule, day); len(courses) > 0 {
				notifyUser(uid, EventReminder, dailyText(courses, day, settings.DailyTomorrow), nil)
			}
		}
	}

	if settings.BeforeClass {
		lead := time.Duration(settings.BeforeMinutes) * time.Minute
		for _, c := range classesOn(schedule, now) {
			startStr, _ := PeriodTimeRange(c.PeriodStart, c.Periods)
			st, err := time.Parse("15:04", startStr)
			if err != nil {
				continue
			}
			start := time.Date(now.Year(), now.Month(), now.Day(), st.Hour(), st.Minute(), 0, 0, cst)
			target := start.Add(-lead)
			if now.Before(target) || !now.Before(start) || now.Sub(target) >= 5*time.Minute {
				continue
			}
			if !markSent(uid, fmt.Sprintf("class:%s:%d", today, c.PeriodStart)) {
				continue
			}
			text := fmt.Sprintf("%d 分钟后上课：%s", settings.BeforeMinutes, describeClass(c))
			if c.Teacher != "" {
				text += "（" + c.Teacher + "）"
			}
			notifyUser(uid, EventReminder, text, nil)
		}
	}
}
