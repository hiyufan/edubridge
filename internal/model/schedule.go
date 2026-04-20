package model

// Course 课程
type Course struct {
	Name       string `json:"name"`
	Teacher    string `json:"teacher"`
	Room       string `json:"room"`
	DayOfWeek  int    `json:"dayOfWeek"`
	PeriodStart int   `json:"periodStart"`
	Periods    int    `json:"periods"`
	Weeks      []int  `json:"weeks,omitempty"`
}

// Schedule 单周课表
type Schedule struct {
	Semester      string   `json:"semester"`
	ClassName     string   `json:"className"`
	StudentName   string   `json:"studentName"`
	Week          int      `json:"week"`           // 教务系统周号 (DQZ)
	CurrentWeek   int      `json:"currentWeek"`    // 真实当前周（根据学期起始日计算）
	SemesterStart string   `json:"semesterStart"`  // 学期起始日 YYYY-MM-DD，从课表HTML日期反推
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
	SemesterStart string   `json:"semesterStart,omitempty"` // 学期起始日 YYYY-MM-DD
	Courses       []Course `json:"courses"`
}

// CaptchaResponse 验证码响应
type CaptchaResponse struct {
	Status   int    `json:"status"`
	Data     string `json:"data"`
	SessionID string `json:"sessionId"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	SessionID string `json:"sessionId"`
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Captcha   string `json:"captcha" binding:"required"`
	LoginType string `json:"loginType"`
}

// ScheduleParams 课表参数
type ScheduleParams struct {
	XN       string
	XQ       int
	DQZ      int
	Sybmdmstr string
	Bjmc     string
}
