package jwclient

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"jww/internal/model"
)

var (
	reTitle       = regexp.MustCompile(`(.+?班)\s*(.+?)同学\s*第(\d+)周\s*课程表\((.+?)\)`)
	reSemester    = regexp.MustCompile(`(\d+-\d+)第(\d+)学期`)
	reScheduleURL = regexp.MustCompile(`xn/([^/]+)/xq/(\d+)/dqz/(\d+)/sybmdmstr/([^/]+)/bjmc/(.+)`)
	reJSURL       = regexp.MustCompile(`['"]\/studentportal\.php\/Jxxx\/xskbxx[^'"]*['"]`)
	reDate        = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
)

var dayNames = map[string]int{
	"星期一": 1, "星期二": 2, "星期三": 3, "星期四": 4,
	"星期五": 5, "星期六": 6, "星期日": 7,
}

// isLoginPage 判断教务系统返回的是否是登录页（会话已失效）。
// 依据：跳转到了登录地址，或页面包含登录表单的提交地址/验证码地址。
func isLoginPage(finalURL, body string) bool {
	if strings.Contains(finalURL, "/Index/login") || strings.HasSuffix(strings.TrimRight(finalURL, "/"), "/studentportal.php") {
		return true
	}
	return strings.Contains(body, "Index/checkLogin") || strings.Contains(body, "Public/verify")
}

func toInt(s string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v
}

// scheduleParams 拼接周课表地址所需的参数
type scheduleParams struct {
	XN        string // 学年
	XQ        int    // 学期
	DQZ       int    // 教务系统认为的当前周
	Sybmdmstr string
	Bjmc      string // 班级名称
}

func (p *scheduleParams) weekPath(week int) string {
	return fmt.Sprintf("/studentportal.php/Jxxx/xskbxx/optype/2/xn/%s/xq/%d/dqz/%d/sybmdmstr/%s/bjmc/%s",
		p.XN, p.XQ, week, p.Sybmdmstr, url.QueryEscape(p.Bjmc))
}

func paramsFromURL(s string) *scheduleParams {
	m := reScheduleURL.FindStringSubmatch(s)
	if len(m) < 6 {
		return nil
	}
	bjmc := m[5]
	if decoded, err := url.QueryUnescape(bjmc); err == nil {
		bjmc = decoded
	}
	return &scheduleParams{XN: m[1], XQ: toInt(m[2]), DQZ: toInt(m[3]), Sybmdmstr: m[4], Bjmc: bjmc}
}

// findScheduleParams 从课表入口页中找到周课表地址参数：依次尝试最终 URL、页面链接、页面脚本
func findScheduleParams(finalURL string, doc *goquery.Document, html string) *scheduleParams {
	if p := paramsFromURL(finalURL); p != nil {
		return p
	}
	var found *scheduleParams
	doc.Find("a[href], iframe[src]").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		href, _ := sel.Attr("href")
		if href == "" {
			href, _ = sel.Attr("src")
		}
		found = paramsFromURL(href)
		return found == nil
	})
	if found != nil {
		return found
	}
	for _, match := range reJSURL.FindAllString(html, -1) {
		if p := paramsFromURL(strings.Trim(match, `"'`)); p != nil {
			return p
		}
	}
	return nil
}

// parseSchedule 解析单周课表页面
func parseSchedule(doc *goquery.Document) *model.Schedule {
	s := &model.Schedule{Week: 1}
	if m := reTitle.FindStringSubmatch(doc.Find(".f2.b").Text()); len(m) >= 5 {
		s.ClassName = strings.TrimSpace(m[1])
		s.StudentName = strings.TrimSpace(m[2])
		s.Week = toInt(m[3])
		s.Semester = strings.TrimSpace(m[4])
	}

	// 表头：每列的星期和日期（格式如 "星期一<br/>2026-03-02"）
	dayHeaders := make([]int, 7)
	dayDates := make([]string, 7)
	doc.Find("table tr").First().Find("td").Each(func(i int, td *goquery.Selection) {
		idx := i - 1
		if i == 0 || idx >= len(dayHeaders) {
			return
		}
		name := strings.TrimSpace(reDate.ReplaceAllString(strings.TrimSpace(td.Text()), ""))
		dayHeaders[idx] = dayNames[name]
		if dayHeaders[idx] == 0 {
			dayHeaders[idx] = i
		}
		// goquery.Text() 会把 <br/> 两边合并，日期从 HTML 中提取
		cellHTML, _ := td.Html()
		if m := reDate.FindStringSubmatch(cellHTML); len(m) > 0 {
			dayDates[idx] = m[1]
		}
	})

	// 学期第 1 周周一 = 本周周一 - (周号-1)*7 天
	if dayDates[0] != "" && s.Week > 0 {
		if monday, err := time.Parse("2006-01-02", dayDates[0]); err == nil {
			s.SemesterStart = monday.AddDate(0, 0, -(s.Week-1)*7).Format("2006-01-02")
		}
	}

	// 课程格子：跨行（rowspan）的格子会占用下面几行同一列
	colOccupied := make(map[int]int)
	doc.Find("table tr").Each(func(ri int, tr *goquery.Selection) {
		if ri == 0 {
			return
		}
		tds := tr.Find("td")
		periodText := tds.First().Text()
		if periodText == "" || strings.Contains(periodText, "中午") {
			return
		}
		periodStart := toInt(strings.NewReplacer("第", "", "节", "").Replace(periodText))

		col := 0
		tds.Each(func(ti int, td *goquery.Selection) {
			if ti == 0 {
				return
			}
			for colOccupied[col] > 0 {
				colOccupied[col]--
				if colOccupied[col] == 0 {
					delete(colOccupied, col)
				}
				col++
			}

			if div := td.Find("div[title]"); div.Length() > 0 {
				title, _ := div.Attr("title")
				parts := strings.Split(title, "\n")
				for i := range parts {
					parts[i] = strings.TrimSpace(parts[i])
				}
				part := func(i int) string {
					if i < len(parts) {
						return parts[i]
					}
					return ""
				}

				rowspan := 1
				if rs, ok := td.Attr("rowspan"); ok {
					rowspan = toInt(rs)
				}
				day := 0
				if col < len(dayHeaders) {
					day = dayHeaders[col]
				}
				if day == 0 {
					day = col + 1
				}

				s.Courses = append(s.Courses, model.Course{
					Name: part(0), Teacher: part(1), Room: part(2),
					DayOfWeek: day, PeriodStart: periodStart, Periods: rowspan,
				})
				if rowspan > 1 {
					colOccupied[col] += rowspan - 1
				}
			}
			col++
		})
	})
	return s
}

// scoreRow 教务系统成绩接口的一行
type scoreRow struct {
	Xn     string `json:"xn"`
	Xq     string `json:"xq"`
	Ssbjmc string `json:"ssbjmc"`
	Kcmc   string `json:"kcmc"`
	Kcxz   string `json:"kcxz"`
	Kcxf   string `json:"kcxf"`
	Zdjsxm string `json:"zdjsxm"`
	Cj     any    `json:"cj"`
	Cjjd   string `json:"cjjd"`
	Cjsx   string `json:"cjsx"`
}

type scorePage struct {
	Total string     `json:"total"` // 教务系统返回的是字符串 "40"
	Rows  []scoreRow `json:"rows"`
}

func (r scoreRow) toScore() model.Score {
	credit, _ := strconv.ParseFloat(r.Kcxf, 64)
	gpa, _ := strconv.ParseFloat(r.Cjjd, 64)
	return model.Score{
		Year: r.Xn, Semester: r.Xq, ClassName: r.Ssbjmc,
		Course: r.Kcmc, Nature: r.Kcxz, Credit: credit,
		Teacher: r.Zdjsxm, Grade: r.Cj, GPA: gpa, Type: r.Cjsx,
	}
}
