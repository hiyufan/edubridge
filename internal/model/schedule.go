// Package model 定义各层共享的纯数据类型，不包含任何逻辑和依赖。
package model

import "time"

// Course 一门课在某个时段的安排（Weeks 为上课周次）
type Course struct {
	Name        string `json:"name"`
	Teacher     string `json:"teacher"`
	Room        string `json:"room"`
	DayOfWeek   int    `json:"dayOfWeek"` // 1-7
	PeriodStart int    `json:"periodStart"`
	Periods     int    `json:"periods"`
	Weeks       []int  `json:"weeks,omitempty"`
}

// Schedule 单周课表
type Schedule struct {
	Semester      string   `json:"semester"`
	ClassName     string   `json:"className"`
	StudentName   string   `json:"studentName"`
	Week          int      `json:"week"`          // 教务系统周号 (DQZ)
	CurrentWeek   int      `json:"currentWeek"`   // 真实当前周（根据学期起始日计算）
	SemesterStart string   `json:"semesterStart"` // 学期起始日 YYYY-MM-DD，从课表日期反推
	Courses       []Course `json:"courses"`
}

// FullSchedule 全学期课表
type FullSchedule struct {
	Semester      string   `json:"semester"`
	ClassName     string   `json:"className"`
	StudentName   string   `json:"studentName"`
	CurrentWeek   int      `json:"currentWeek"`
	TotalWeeks    int      `json:"totalWeeks"`
	FetchedWeeks  int      `json:"fetchedWeeks"`
	SemesterStart string   `json:"semesterStart,omitempty"`
	Courses       []Course `json:"courses"`
}

// Occurrence 某一周的一次具体上课
type Occurrence struct {
	Week        int    `json:"week"`
	DayOfWeek   int    `json:"dayOfWeek"`
	PeriodStart int    `json:"periodStart"`
	Periods     int    `json:"periods"`
	Name        string `json:"name"`
	Teacher     string `json:"teacher"`
	Room        string `json:"room"`
}

// OccurrenceChange 同一节课的教室/老师/节数变化
type OccurrenceChange struct {
	Old Occurrence `json:"old"`
	New Occurrence `json:"new"`
}

// ScheduleDiff 课表变动（只比较当前周及以后）
type ScheduleDiff struct {
	Added      []Occurrence       `json:"added"`   // 加课
	Removed    []Occurrence       `json:"removed"` // 减课/停课
	Changed    []OccurrenceChange `json:"changed"` // 换教室、换老师等
	DetectedAt time.Time          `json:"detectedAt"`
}

// Empty 是否没有任何变动
func (d *ScheduleDiff) Empty() bool {
	return d == nil || len(d.Added)+len(d.Removed)+len(d.Changed) == 0
}

// ConflictPair 课程冲突对
type ConflictPair struct {
	CourseA       Course `json:"courseA"`
	CourseB       Course `json:"courseB"`
	ConflictWeeks []int  `json:"conflictWeeks"`
}
