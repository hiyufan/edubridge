package model

// Score 成绩
type Score struct {
	Year     string  `json:"year"`
	Semester string  `json:"semester"`
	ClassName string `json:"className"`
	Course   string  `json:"course"`
	Nature   string  `json:"nature"`
	Credit   float64 `json:"credit"`
	Teacher  string  `json:"teacher"`
	Grade    any     `json:"grade"`
	GPA      float64 `json:"gpa"`
	Type     string  `json:"type"`
}

// ScoreStats 成绩统计
type ScoreStats struct {
	TotalCredits  float64          `json:"totalCredits"`  // 总学分
	WeightedGPA   float64         `json:"weightedGPA"`   // 加权GPA
	SimpleGPA     float64          `json:"simpleGPA"`     // 算术GPA
	FailedCount   int             `json:"failedCount"`    // 挂科数
	TotalCourses  int             `json:"totalCourses"`   // 总课程数
	SemesterStats []SemesterStat  `json:"semesterStats"` // 逐学期统计
}

// SemesterStat 学期统计
type SemesterStat struct {
	Semester     string  `json:"semester"`     // 学期名称
	Year         string  `json:"year"`         // 学年
	Term         int     `json:"term"`         // 学期 1或2
	Credits      float64 `json:"credits"`      // 学期学分
	GPA          float64 `json:"gpa"`          // 学期GPA
	CourseCount  int     `json:"courseCount"`  // 课程数
	FailedCount  int     `json:"failedCount"` // 挂科数
}

// GPAConfig GPA 配置
type GPAConfig struct {
	TopCredit     float64 `json:"topCredit"`     // 最高学分
	SimulateGPA   float64 `json:"simulateGPA"`  // 模拟GPA
}
