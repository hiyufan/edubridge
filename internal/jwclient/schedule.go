package jwclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"jww/internal/model"
)

const entryPath = "/studentportal.php/Jxxx/xskbxx/optype/1"

// scheduleEntry 课表入口页：要么直接就是一张周课表，要么包含周课表地址参数
type scheduleEntry struct {
	params   *scheduleParams
	schedule *model.Schedule // 入口页本身就是课表时非空
}

func (c *Client) scheduleEntry(ctx context.Context) (*scheduleEntry, error) {
	p, err := c.get(ctx, entryPath)
	if err != nil {
		return nil, err
	}
	html := string(p.body)
	isSchedule := strings.Contains(html, "课程表")
	if !isSchedule && isLoginPage(p.finalURL, html) {
		return nil, ErrSessionExpired
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("%w: 课表入口页解析失败", ErrUnavailable)
	}
	if isSchedule && reTitle.MatchString(doc.Find(".f2.b").Text()) {
		return &scheduleEntry{schedule: parseSchedule(doc)}, nil
	}
	if params := findScheduleParams(p.finalURL, doc, html); params != nil {
		return &scheduleEntry{params: params}, nil
	}
	return nil, fmt.Errorf("%w: 无法获取课表参数", ErrUnavailable)
}

// weekPage 获取某一周的课表；ok=false 表示该周没有课表页（例如超出学期范围）
func (c *Client) weekPage(ctx context.Context, params *scheduleParams, week int) (s *model.Schedule, ok bool, err error) {
	p, err := c.get(ctx, params.weekPath(week))
	if err != nil {
		return nil, false, err
	}
	if p.status != http.StatusOK {
		return nil, false, fmt.Errorf("%w: 第%d周课表 HTTP %d", ErrUnavailable, week, p.status)
	}
	html := string(p.body)
	if !strings.Contains(html, "课程表") {
		if isLoginPage(p.finalURL, html) {
			return nil, false, ErrSessionExpired
		}
		return nil, false, nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, false, fmt.Errorf("%w: 第%d周课表解析失败", ErrUnavailable, week)
	}
	return parseSchedule(doc), true, nil
}

// WeekSchedule 获取单周课表；week<=0 时取教务系统默认的那一周。
// 入口页本身就是课表时直接返回入口页的那一周。
func (c *Client) WeekSchedule(ctx context.Context, week int) (*model.Schedule, error) {
	entry, err := c.scheduleEntry(ctx)
	if err != nil {
		return nil, err
	}
	if entry.schedule != nil {
		return entry.schedule, nil
	}
	if week <= 0 {
		week = entry.params.DQZ
	}
	s, ok, err := c.weekPage(ctx, entry.params, week)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &model.Schedule{Week: week}, nil
	}
	return s, nil
}

// FullSchedule 并发拉取第 1..maxWeek 周课表并合并成全学期课表。
// complete 表示每一周都成功拉取；不完整的结果不应用来判断“减课”。
func (c *Client) FullSchedule(ctx context.Context, maxWeek int) (result *model.FullSchedule, complete bool, err error) {
	entry, err := c.scheduleEntry(ctx)
	if err != nil {
		return nil, false, err
	}

	// 入口页直接是课表：只能拿到这一周
	if entry.schedule != nil {
		s := entry.schedule
		courses := make([]model.Course, len(s.Courses))
		for i, course := range s.Courses {
			course.Weeks = []int{s.Week}
			courses[i] = course
		}
		return &model.FullSchedule{
			Semester: s.Semester, ClassName: s.ClassName, StudentName: s.StudentName, CurrentWeek: s.Week,
			TotalWeeks: maxWeek, FetchedWeeks: 1, SemesterStart: s.SemesterStart, Courses: courses,
		}, false, nil
	}

	type key struct {
		name, teacher string
		day, period   int
	}
	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		merged   = make(map[key]*model.Course)
		info     model.FullSchedule
		failed   int
		expired  bool
		sem      = make(chan struct{}, min(maxWeek/3+1, 5))
		semester string
	)
	for w := 1; w <= maxWeek; w++ {
		wg.Add(1)
		go func(week int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			s, ok, err := c.weekPage(ctx, entry.params, week)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case errors.Is(err, ErrSessionExpired):
				expired = true
				failed++
				return
			case err != nil:
				failed++
				return
			case !ok:
				return
			}

			info.FetchedWeeks++
			if semester == "" {
				semester = s.Semester
				info.ClassName = s.ClassName
				info.StudentName = s.StudentName
			}
			if info.SemesterStart == "" {
				info.SemesterStart = s.SemesterStart
			}
			for _, course := range s.Courses {
				k := key{course.Name, course.Teacher, course.DayOfWeek, course.PeriodStart}
				if existing, ok := merged[k]; ok {
					existing.Weeks = append(existing.Weeks, week)
				} else {
					c := course
					c.Weeks = []int{week}
					merged[k] = &c
				}
			}
		}(w)
	}
	wg.Wait()

	if expired {
		return nil, false, ErrSessionExpired
	}

	courses := make([]model.Course, 0, len(merged))
	for _, course := range merged {
		sort.Ints(course.Weeks)
		courses = append(courses, *course)
	}
	sort.Slice(courses, func(i, j int) bool {
		a, b := courses[i], courses[j]
		if a.DayOfWeek != b.DayOfWeek {
			return a.DayOfWeek < b.DayOfWeek
		}
		if a.PeriodStart != b.PeriodStart {
			return a.PeriodStart < b.PeriodStart
		}
		return a.Name < b.Name
	})

	info.Semester = semester
	info.CurrentWeek = entry.params.DQZ // 教务系统认为的当前周，调用方可按学期起始日校正
	info.TotalWeeks = maxWeek
	info.Courses = courses
	return &info, failed == 0, nil
}
