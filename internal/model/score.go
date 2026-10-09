package model

// Score 一门课的成绩
type Score struct {
	Year      string  `json:"year"`
	Semester  string  `json:"semester"`
	ClassName string  `json:"className"`
	Course    string  `json:"course"`
	Nature    string  `json:"nature"`
	Credit    float64 `json:"credit"`
	Teacher   string  `json:"teacher"`
	Grade     any     `json:"grade"` // 数字或“优秀”等文字
	GPA       float64 `json:"gpa"`
	Type      string  `json:"type"`
}

// ScoreChange 新出或被修改的成绩
type ScoreChange struct {
	Score    Score  `json:"score"`
	OldGrade string `json:"oldGrade,omitempty"` // 非空表示成绩被修改
}

// SemesterStat 学期统计
type SemesterStat struct {
	Semester    string  `json:"semester"`
	Year        string  `json:"year"`
	Term        string  `json:"term"`
	Credits     float64 `json:"credits"`
	GPA         float64 `json:"gpa"`
	CourseCount int     `json:"courseCount"`
	FailedCount int     `json:"failedCount"`
}

// ScoreStats 成绩统计
type ScoreStats struct {
	TotalCredits  float64        `json:"totalCredits"`
	WeightedGPA   float64        `json:"weightedGPA"`
	SimpleGPA     float64        `json:"simpleGPA"`
	FailedCount   int            `json:"failedCount"`
	TotalCourses  int            `json:"totalCourses"`
	SemesterStats []SemesterStat `json:"semesterStats"`
	HighestCourse *Score         `json:"highestCourse,omitempty"`
	LowestCourse  *Score         `json:"lowestCourse,omitempty"`
	FailedCourses []Score        `json:"failedCourses"`
}

// KnownScores 已经见过的成绩，用于发现新成绩
type KnownScores struct {
	Complete bool              `json:"complete"` // 是否已经有过一次完整拉取（基线可用）
	Grades   map[string]string `json:"grades"`   // 学年|学期|课程 -> 成绩
}
