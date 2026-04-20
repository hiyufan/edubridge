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
