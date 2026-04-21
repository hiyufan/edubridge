package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"jww/internal/model"
)

// C5 修复：魔法数字集中管理（service 层）
const (
	serviceSessionTTL       = 30 * time.Minute // 会话过期时间
	serviceScoreCacheTTL    = 15 * time.Minute // 成绩缓存过期时间
	serviceScheduleCacheTTL = 5  * time.Minute // 课表缓存过期时间
)

const baseURL = "https://jw.fzrjxy.com"

var (
	jwService *JwService
	once      sync.Once
)

// 预编译的正则表达式
var (
	reTitle       = regexp.MustCompile(`(.+?班)\s*(.+?)同学\s*第(\d+)周\s*课程表\((.+?)\)`)
	reSemester    = regexp.MustCompile(`(\d+-\d+)第(\d+)学期`)
	reScheduleURL = regexp.MustCompile(`xn/([^/]+)/xq/(\d+)/dqz/(\d+)/sybmdmstr/([^/]+)/bjmc/(.+)`)
	reJSURL       = regexp.MustCompile(`['"]\/studentportal\.php\/Jxxx\/xskbxx[^'"]*['"]`)
	reDate        = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`) // B6: 提升包级避免重复编译
)

// GetJwService 获取单例
func GetJwService() *JwService {
	once.Do(func() {
		jwService = NewJwService()
	})
	return jwService
}

// Session 用户会话
type Session struct {
	Client      *resty.Client
	HttpClient  *http.Client
	CookieJar   *jar
	UID         string
	ExpireTime  time.Time
	ScoreCache  *ScoreCache
}

// ScoreCache 成绩缓存
type ScoreCache struct {
	Data   []model.Score
	Expire time.Time
}

// JwService 教务服务
type JwService struct {
	sessions      map[string]*Session
	mu            sync.RWMutex
	scheduleCache map[string]*ScheduleCache
	stopCh        chan struct{}
}

type ScheduleCache struct {
	Data   *model.FullSchedule
	Expire time.Time
}

// NewJwService 创建服务实例
func NewJwService() *JwService {
	svc := &JwService{
		sessions:      make(map[string]*Session),
		scheduleCache: make(map[string]*ScheduleCache),
		stopCh:        make(chan struct{}),
	}
	go svc.cleanup()
	return svc
}

// Close 停止后台 goroutine
func (s *JwService) Close() {
	close(s.stopCh)
}

// cleanup 定期清理过期会话（BUG-6 修复：有退出 channel）
func (s *JwService) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for id, session := range s.sessions {
				if now.After(session.ExpireTime) {
					delete(s.sessions, id)
				}
			}
			for id, cache := range s.scheduleCache {
				if now.After(cache.Expire) {
					delete(s.scheduleCache, id)
				}
			}
			s.mu.Unlock()
		}
	}
}

// getSession 获取或创建会话（C8 优化：缩小锁粒度，读锁检查→写锁创建的双重检查模式）
func (s *JwService) getSession(sessionID string) *Session {
	// 第一次尝试：读锁检查（大多数情况 session 已存在且未过期）
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	if exists && !time.Now().After(session.ExpireTime) {
		s.mu.RUnlock()
		return session
	}
	s.mu.RUnlock()

	// 需要创建新 session，升级为写锁
	s.mu.Lock()
	defer s.mu.Unlock()

	// 双重检查：持有写锁后再次确认（其他 goroutine 可能已创建）
	session, exists = s.sessions[sessionID]
	if !exists || time.Now().After(session.ExpireTime) {
		cookieJar := &jar{}
		client := resty.New()
		client.SetBaseURL(baseURL)
		client.SetTimeout(20 * time.Second)
		client.SetCookieJar(cookieJar)
		client.SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		client.SetHeader("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
		client.SetHeader("Connection", "keep-alive")
		client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(10))

		httpClient := &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 20 * time.Second,
			},
			Timeout: 20 * time.Second,
			Jar:     cookieJar,
		}

		session = &Session{
			Client:     client,
			HttpClient: httpClient,
			CookieJar:  cookieJar,
			ExpireTime: time.Now().Add(serviceSessionTTL),
		}
		s.sessions[sessionID] = session
	}
	return session
}

// checkSession 验证会话是否有效
func (s *JwService) checkSession(sessionID string) (*Session, error) {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists || time.Now().After(session.ExpireTime) || session.UID == "" {
		return nil, fmt.Errorf("登录已过期，请重新登录")
	}
	return session, nil
}

// FetchCaptcha 获取验证码
func (s *JwService) FetchCaptcha(sessionID string) ([]byte, string, error) {
	session := s.getSession(sessionID)

	// 用 session 的 http.Client（共享 cookie jar）
	req, err := http.NewRequest("GET", baseURL+"/studentportal.php/Public/verify/", nil)
	if err != nil {
		return nil, "", fmt.Errorf("构建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/png,image/*;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Referer", baseURL+"/")

	resp, err := session.HttpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("获取验证码失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("读取验证码失败: %v", err)
	}

	if resp.StatusCode != 200 || len(body) == 0 {
		return nil, "", fmt.Errorf("验证码获取失败 status=%d bodylen=%d", resp.StatusCode, len(body))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}

	return body, contentType, nil
}

// Login 登录
func (s *JwService) Login(sessionID, username, password, captcha, loginType string) error {
	session := s.getSession(sessionID)

	loginTypeField := "xsxh"
	switch loginType {
	case "zjh":
		loginTypeField = "zjh"
	case "gkksh":
		loginTypeField = "gkksh"
	}

	// MD5 password
	hash := md5.Sum([]byte(password))
	dlmm := hex.EncodeToString(hash[:])

	// 用 session 的 http.Client（共享 cookie jar）
	postData := fmt.Sprintf("logintype=%s&%s=%s&dlmm=%s&yzm=%s",
		url.QueryEscape(loginType),
		loginTypeField, url.QueryEscape(username),
		dlmm, url.QueryEscape(captcha))

	req, err := http.NewRequest("POST", baseURL+"/studentportal.php/Index/checkLogin", strings.NewReader(postData))
	if err != nil {
		return fmt.Errorf("构建登录请求失败: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json,text/html,*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Referer", baseURL+"/")

	resp, err := session.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("登录请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取登录响应失败: %v", err)
	}

	var result loginResp
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("登录响应解析失败: %v, body: %s", err, string(body))
	}

	if result.Status != 1 {
		return fmt.Errorf("%s", result.Info)
	}

	// 访问 gotourl 建立完整 session（NB1 修复：resp.Body 必须关闭）
	if result.Gotourl != "" {
		req, err := http.NewRequest("GET", result.Gotourl, nil)
		if err == nil {
			resp, err := session.HttpClient.Do(req)
			if err != nil {
				slog.Warn("gotourl 跳转失败", "url", result.Gotourl, "err", err)
			} else {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	} else {
		req, err := http.NewRequest("GET", baseURL+"/studentportal.php/Main/", nil)
		if err == nil {
			resp, err := session.HttpClient.Do(req)
			if err != nil {
				slog.Warn("Main 页面访问失败", "err", err)
			} else {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	}

	s.mu.Lock()
	session.UID = username
	session.ExpireTime = time.Now().Add(serviceSessionTTL)
	s.mu.Unlock()

	return nil
}

type loginResp struct {
	Status  int    `json:"status"`
	Info    string `json:"info"`
	Gotourl string `json:"gotourl"`
}

// GetSchedulePage 获取单周课表 HTML
func (s *JwService) GetSchedulePage(sessionID string, week *int) (string, error) {
	session, err := s.checkSession(sessionID)
	if err != nil {
		return "", err
	}

	doc, params, err := s.fetchScheduleParams(session)
	if err != nil {
		return "", err
	}
	_ = doc // EntryHtml 非空时 params.EntryHtml 已含 HTML，无需额外处理

	// 如果入口页直接是课表
	if params.EntryHtml != "" {
		return params.EntryHtml, nil
	}

	targetWeek := params.DQZ
	if week != nil {
		targetWeek = *week
	}

	scheduleURL := fmt.Sprintf(
		"/studentportal.php/Jxxx/xskbxx/optype/2/xn/%s/xq/%d/dqz/%d/sybmdmstr/%s/bjmc/%s",
		params.XN, params.XQ, targetWeek, params.Sybmdmstr, url.QueryEscape(params.Bjmc),
	)

	resp, err := session.Client.R().Get(scheduleURL)
	if err != nil {
		return "", fmt.Errorf("获取课表失败: %v", err)
	}

	return resp.String(), nil
}

// ScheduleParams 课表参数
type ScheduleParams struct {
	XN        string
	XQ        int
	DQZ       int
	Sybmdmstr string
	Bjmc      string
	EntryHtml string
}

// fetchScheduleParams 获取课表参数（可选返回已解析的 goquery.Document）
func (s *JwService) fetchScheduleParams(session *Session) (*goquery.Document, *ScheduleParams, error) {
	resp, err := session.Client.R().Get("/studentportal.php/Jxxx/xskbxx/optype/1")
	if err != nil {
		return nil, nil, fmt.Errorf("获取课表入口失败: %v", err)
	}

	html := resp.String()

	// 情况1：入口页直接是课表
	if strings.Contains(html, "课程表") {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			return nil, nil, fmt.Errorf("HTML 解析失败: %w", err)
		}
		title := doc.Find(".f2.b").Text()

		matches := reTitle.FindStringSubmatch(title)

		if len(matches) >= 5 {
			semMatch := reSemester.FindStringSubmatch(matches[4])
			xn := ""
			xq := 1
			if len(semMatch) >= 3 {
				xn = semMatch[1]
				xq, _ = strconv.Atoi(semMatch[2])
			}

			return doc, &ScheduleParams{
				XN:        xn,
				XQ:        xq,
				DQZ:       toInt(matches[3]),
				Bjmc:      matches[1],
				EntryHtml: html,
			}, nil
		}
	}

	// 尝试从 URL 提取参数
	if params := extractParamsFromURL(resp.Request.URL); params != nil {
		return nil, params, nil
	}

	// 尝试从 HTML 中的链接提取（使用闭包捕获结果）
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, nil, fmt.Errorf("HTML 解析失败: %w", err)
	}
	var foundParams *ScheduleParams
	doc.Find("a[href], iframe[src]").EachWithBreak(func(i int, sel *goquery.Selection) bool {
		href, _ := sel.Attr("href")
		if href == "" {
			href, _ = sel.Attr("src")
		}
		if p := extractParamsFromURL(href); p != nil {
			foundParams = p
			return false // 停止遍历
		}
		return true
	})
	if foundParams != nil {
		return doc, foundParams, nil
	}

	// 从 JS 代码中提取
	for _, match := range reJSURL.FindAllString(html, -1) {
		cleaned := strings.Trim(match, `"'`)
		if params := extractParamsFromURL(cleaned); params != nil {
			return nil, params, nil
		}
	}

	return nil, nil, fmt.Errorf("无法获取课表参数")
}

func extractParamsFromURL(urlStr string) *ScheduleParams {
	matches := reScheduleURL.FindStringSubmatch(urlStr)
	if len(matches) < 6 {
		return nil
	}

	xq, _ := strconv.Atoi(matches[2])
	dqz, _ := strconv.Atoi(matches[3])

	return &ScheduleParams{
		XN:        matches[1],
		XQ:        xq,
		DQZ:       dqz,
		Sybmdmstr: matches[4],
		Bjmc:      decodeURL(matches[5]),
	}
}

func decodeURL(s string) string {
	if decoded, err := url.QueryUnescape(s); err == nil {
		return decoded
	}
	return s
}

func toInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

// ParseSchedule 解析课表 HTML
func (s *JwService) ParseSchedule(html string) (*model.Schedule, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("解析课表失败: %v", err)
	}
	return s.parseScheduleDoc(doc)
}

// parseScheduleDoc 从已解析的 goquery.Document 提取课表数据（P5 修复：避免重复解析同一 HTML）
func (s *JwService) parseScheduleDoc(doc *goquery.Document) (*model.Schedule, error) {
	title := doc.Find(".f2.b").Text()

	// 解析标题
	matches := reTitle.FindStringSubmatch(title)

	className := ""
	studentName := ""
	week := 1
	semester := ""

	if len(matches) >= 5 {
		className = strings.TrimSpace(matches[1])
		studentName = strings.TrimSpace(matches[2])
		week = toInt(matches[3])
		semester = strings.TrimSpace(matches[4])
	}

	dayMap := map[string]int{
		"星期一": 1, "星期二": 2, "星期三": 3, "星期四": 4,
		"星期五": 5, "星期六": 6, "星期日": 7,
	}

	var courses []model.Course
	colOccupied := make(map[int]int)
	dayHeaders := make([]int, 7)
	dayDates := make([]string, 7) // 存储每天的日期 "2026-04-13"
	_ = reDate                     // 引用包级变量，避免编译器优化移除

	// 解析表头（包含日期）
	doc.Find("table tr").First().Find("td").Each(func(i int, td *goquery.Selection) {
		if i == 0 {
			return
		}
		idx := i - 1
		if idx >= len(dayHeaders) {
			return
		}
		cellText := td.Text()
		dayName := strings.TrimSpace(cellText)
		// 去掉日期部分得到星期名（格式如 "星期一2026-03-02" 或 "星期一\n2026-03-02"）
		dayName = reDate.ReplaceAllString(dayName, "")
		dayName = strings.TrimSpace(dayName)
		dayHeaders[idx] = dayMap[dayName]
		if dayHeaders[idx] == 0 {
			dayHeaders[idx] = i
		}
		// 从 HTML 中提取日期（goquery.Text() 会把 <br/> 两边内容合并，需从 HTML 正则提取）
		cellHtml, _ := td.Html()
		matches := reDate.FindStringSubmatch(cellHtml)
		if len(matches) > 0 {
			dayDates[idx] = matches[1]
		}
	})

	// 从课表HTML中的实际日期 + DQZ 反推学期起始日
	// 原理：dayDates[0] 是当前DQZ周的周一日期
	//       学期第1周的周一 = dayDates[0] - (DQZ-1)*7天
	semesterStart := ""
	if dayDates[0] != "" && week > 0 {
		if firstDate, err := time.Parse("2006-01-02", dayDates[0]); err == nil {
			start := firstDate.AddDate(0, 0, -(week-1)*7)
			semesterStart = start.Format("2006-01-02")
		}
	}

	// 解析课程
	doc.Find("table tr").Each(func(ri int, tr *goquery.Selection) {
		if ri == 0 {
			return
		}

		tds := tr.Find("td")
		periodText := tds.First().Text()
		if periodText == "" || strings.Contains(periodText, "中午") {
			return
		}

		periodStart := toInt(strings.ReplaceAll(strings.ReplaceAll(periodText, "第", ""), "节", ""))

		colIdx := 0
		tds.Each(func(ti int, td *goquery.Selection) {
			if ti == 0 {
				return
			}

			for colOccupied[colIdx] > 0 {
				colOccupied[colIdx]--
				if colOccupied[colIdx] == 0 {
					delete(colOccupied, colIdx)
				}
				colIdx++
			}

			div := td.Find("div[title]")
			if div.Length() > 0 {
				titleStr, _ := div.Attr("title")
				parts := strings.Split(titleStr, "\n")
				for i := range parts {
					parts[i] = strings.TrimSpace(parts[i])
				}

				rowspan := 1
				if rs, ok := td.Attr("rowspan"); ok {
					rowspan = toInt(rs)
				}

				dayOfWeek := 0
				if colIdx < len(dayHeaders) {
					dayOfWeek = dayHeaders[colIdx]
				}
				if dayOfWeek == 0 {
					dayOfWeek = colIdx + 1
				}

				courses = append(courses, model.Course{
					Name:        getOrEmpty(parts, 0),
					Teacher:     getOrEmpty(parts, 1),
					Room:        getOrEmpty(parts, 2),
					DayOfWeek:   dayOfWeek,
					PeriodStart: periodStart,
					Periods:     rowspan,
				})

				if rowspan > 1 {
					// B10 修复：累加而非覆盖，保留之前同一列其他行占用的 rowspan
					colOccupied[colIdx] += rowspan - 1
				}
			}
			colIdx++
		})
	})

	return &model.Schedule{
		Semester:      semester,
		ClassName:     className,
		StudentName:   studentName,
		Week:          week,
		SemesterStart: semesterStart,
		Courses:       courses,
	}, nil
}

func getOrEmpty(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return ""
}

// CalcRealCurrentWeek 根据学期起始日和今天日期计算真实当前周
func CalcRealCurrentWeek(semesterStart string, maxWeek int) int {
	start, err := time.Parse("2006-01-02", semesterStart)
	if err != nil {
		return 1
	}
	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())

	daysDiff := int(today.Sub(start).Hours() / 24)
	week := daysDiff/7 + 1
	if week < 1 {
		week = 1
	}
	if week > maxWeek {
		week = maxWeek
	}
	return week
}

// GetFullSchedule 获取全学期课表（并发）
func (s *JwService) GetFullSchedule(sessionID string, maxWeek int) (*model.FullSchedule, error) {
	// 检查缓存（读锁范围覆盖整个深拷贝过程，防止返回值在锁外被并发修改）
	s.mu.RLock()
	cache, ok := s.scheduleCache[sessionID]
	if ok && time.Now().Before(cache.Expire) {
		// 深拷贝：防止返回的指针在锁释放后与写操作产生 data race
		result := *cache.Data
		result.Courses = make([]model.Course, len(cache.Data.Courses))
		copy(result.Courses, cache.Data.Courses)
		s.mu.RUnlock()
		return &result, nil
	}
	s.mu.RUnlock()

	session, err := s.checkSession(sessionID)
	if err != nil {
		return nil, err
	}

	doc, params, err := s.fetchScheduleParams(session)
	if err != nil {
		return nil, err
	}

	var result *model.FullSchedule

	// 入口页直接是课表时，只返回当前周
	if params.EntryHtml != "" {
		parsed, _ := s.parseScheduleDoc(doc)
		weeks := make([]int, 1)
		weeks[0] = params.DQZ

		// 计算真实当前周
		realCurrentWeek := params.DQZ
		if parsed.SemesterStart != "" {
			realCurrentWeek = CalcRealCurrentWeek(parsed.SemesterStart, maxWeek)
		}

		result = &model.FullSchedule{
			Semester:      parsed.Semester,
			ClassName:     parsed.ClassName,
			StudentName:   parsed.StudentName,
			CurrentWeek:   realCurrentWeek,
			TotalWeeks:    maxWeek,
			FetchedWeeks:  1,
			SemesterStart: parsed.SemesterStart,
			Courses:       make([]model.Course, len(parsed.Courses)),
		}
		for i, c := range parsed.Courses {
			result.Courses[i] = c
			result.Courses[i].Weeks = weeks
		}
	} else {
		courseMap := make(map[string]*courseWithWeeks) // 普通 map，用 mutex 保护
		var mu sync.Mutex
		fetchedWeeks := 0
		currentWeek := params.DQZ
		semester := ""
		className := ""
		studentName := ""
		semesterStart := ""

		var wg sync.WaitGroup
		concurrency := maxWeek/3 + 1
		if concurrency > 5 {
			concurrency = 5
		}
		sem := make(chan struct{}, concurrency) // 并发控制

		for start := 1; start <= maxWeek; start++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				scheduleURL := fmt.Sprintf(
					"/studentportal.php/Jxxx/xskbxx/optype/2/xn/%s/xq/%d/dqz/%d/sybmdmstr/%s/bjmc/%s",
					params.XN, params.XQ, w, params.Sybmdmstr, url.QueryEscape(params.Bjmc),
				)

				// B3 修复：不使用共享的 resty.Client（并发不安全），改用 session.HttpClient
				req, err := http.NewRequest("GET", baseURL+scheduleURL, nil)
				if err != nil {
					return
				}
				req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
				req.Header.Set("Accept", "text/html,*/*")
				req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
				req.Header.Set("Referer", baseURL+"/")

				resp, err := session.HttpClient.Do(req)
				if err != nil || resp == nil {
					return
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return
				}

				if !strings.Contains(string(body), "课程表") {
					return
				}

				parsed, err := s.ParseSchedule(string(body))
				if err != nil || parsed == nil {
					return
				}

				mu.Lock()
				fetchedWeeks++
				if semester == "" {
					semester = parsed.Semester
				}
				if className == "" {
					className = parsed.ClassName
				}
				if studentName == "" {
					studentName = parsed.StudentName
				}
				if semesterStart == "" && parsed.SemesterStart != "" {
					semesterStart = parsed.SemesterStart
				}
				mu.Unlock()

				for _, c := range parsed.Courses {
					key := fmt.Sprintf("%s|%s|%d|%d", c.Name, c.Teacher, c.DayOfWeek, c.PeriodStart)
					if existing, ok := courseMap[key]; ok {
						existing.weeks = append(existing.weeks, w)
					} else {
						courseMap[key] = &courseWithWeeks{course: c, weeks: []int{w}}
					}
				}
			}(start)
		}

		wg.Wait()

		mu.Lock()
		var courses []model.Course
		for _, entry := range courseMap {
			entry.course.Weeks = entry.weeks
			courses = append(courses, entry.course)
		}
		mu.Unlock()

		// 计算真实当前周
		realCurrentWeek := currentWeek
		if semesterStart != "" {
			realCurrentWeek = CalcRealCurrentWeek(semesterStart, maxWeek)
		}

		result = &model.FullSchedule{
			Semester:      semester,
			ClassName:     className,
			StudentName:   studentName,
			CurrentWeek:   realCurrentWeek,
			TotalWeeks:    maxWeek,
			FetchedWeeks:  fetchedWeeks,
			SemesterStart: semesterStart,
			Courses:       courses,
		}
	}

	// 存入缓存
	s.mu.Lock()
	s.scheduleCache[sessionID] = &ScheduleCache{
		Data:   result,
		Expire: time.Now().Add(serviceScheduleCacheTTL),
	}
	s.mu.Unlock()

	return result, nil
}

type courseWithWeeks struct {
	course model.Course
	weeks  []int
}

// GetCachedSemesters 从成绩缓存中提取学期列表（P4 修复）
func (s *JwService) GetCachedSemesters(sessionID string) ([]string, error) {
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session 不存在")
	}

	s.mu.RLock()
	cache := session.ScoreCache
	s.mu.RUnlock()
	if cache == nil || time.Now().After(cache.Expire) {
		return nil, fmt.Errorf("缓存不存在或已过期")
	}

	semesterSet := make(map[string]struct{})
	for _, sc := range cache.Data {
		semester := sc.Year + "-" + sc.Semester
		semesterSet[semester] = struct{}{}
	}

	semesters := make([]string, 0, len(semesterSet))
	for k := range semesterSet {
		semesters = append(semesters, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(semesters)))
	return semesters, nil
}

// GetScorePage 获取成绩（支持按学期过滤）
func (s *JwService) GetScorePage(sessionID, semester string) ([]model.Score, error) {
	session, err := s.checkSession(sessionID)
	if err != nil {
		return nil, err
	}

	// 尝试从缓存获取全部成绩（NB3 修复：读 ScoreCache 需要 s.mu 保护）
	s.mu.RLock()
	cache := session.ScoreCache
	s.mu.RUnlock()
	if semester == "" && cache != nil && time.Now().Before(cache.Expire) {
		return cache.Data, nil
	}

	// B8 修复：删除 xn/xq 死代码（原来声明后仅用于 slog，未参与过滤）
	slog.Info("GetScorePage 请求参数", "semester", semester)

	// 循环分页拿完全部成绩
	// 教务系统忽略 rows 参数，硬编码每页最多 9 条，必须循环分页
	allScores, err := s.fetchAllScores(session)
	if err != nil {
		return nil, err
	}

	slog.Info("成绩获取完成", "total_collected", len(allScores))

	// 缓存全部未过滤的成绩（BUG-7 修复：在过滤前缓存，NB3 修复：写 ScoreCache 需要 s.mu 保护）
	s.mu.Lock()
	session.ScoreCache = &ScoreCache{
		Data:   allScores,
		Expire: time.Now().Add(serviceScoreCacheTTL),
	}
	s.mu.Unlock()

	// 客户端按学期过滤（xn+xq 组合精确匹配）
	if semester != "" {
		filtered := []model.Score{}
		for _, sc := range allScores {
			if sc.Year+"-"+sc.Semester == semester {
				filtered = append(filtered, sc)
			}
		}
		slog.Info("学期过滤后", "origin", len(allScores), "filtered", len(filtered), "target", semester)
		return filtered, nil
	}

	return allScores, nil
}

// buildScoreRequest 构造带有通用 Header 的 HTTP 请求（C2 优化：消除重复 Header 代码）
func buildScoreRequest(method, url string, body io.Reader, contentType string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json,text/html,*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Referer", baseURL+"/")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

// fetchAllScores 分页并发拉取全部成绩
func (s *JwService) fetchAllScores(session *Session) ([]model.Score, error) {
	seen := sync.Map{}

	// 第一页：获取总数
	pageSize := 9
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	queryParams := map[string]string{
		"page":  "1", "rows": strconv.Itoa(pageSize), "start": "0",
		"p":     "1", "pn":    "1", "_": ts,
	}
	req, err := buildScoreRequest("GET", baseURL+"/studentportal.php/Jxxx/cjxxlb", nil, "")
	if err != nil {
		return nil, fmt.Errorf("构建成绩请求失败: %v", err)
	}
	q := req.URL.Query()
	for k, v := range queryParams { q.Add(k, v) }
	req.URL.RawQuery = q.Encode()

	resp, err := session.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取成绩失败: %v", err)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("读取成绩响应失败: %v", err)
	}

	var firstResp scorePageResp
	if err := json.Unmarshal(bodyBytes, &firstResp); err != nil {
		return nil, fmt.Errorf("解析成绩失败: %v", err)
	}

	totalRows := 0
	if firstResp.Total != "" {
		totalRows, _ = strconv.Atoi(firstResp.Total)
	}
	// totalRows 为 0 或 total 为空字符串（教务系统无成绩）时，直接解析第一页后返回
	if totalRows <= 0 {
		return parseScoreRows(firstResp.Rows, &seen), nil
	}
	totalPages := (totalRows + pageSize - 1) / pageSize

	// 后续页并发抓取
	concurrency := 5
	if concurrency > totalPages-1 {
		concurrency = totalPages - 1
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]scorePageResp, totalPages)
	failed := make([]bool, totalPages) // 标记哪些页失败

	// 第一页结果先放入
	results[0] = firstResp

	for page := 2; page <= totalPages; page++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			start := (p - 1) * pageSize
			ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
			postData := url.Values{
				"page":  {strconv.Itoa(p)},
				"rows":  {strconv.Itoa(pageSize)},
				"start": {strconv.Itoa(start)},
				"p":     {strconv.Itoa(p)},
				"pn":    {strconv.Itoa(p)},
				"_":     {ts},
			}
			postReq, err := buildScoreRequest("POST", baseURL+"/studentportal.php/Jxxx/cjxxlb",
				strings.NewReader(postData.Encode()), "application/x-www-form-urlencoded")
			if err != nil {
				mu.Lock()
				failed[p-1] = true
				slog.Warn("构建成绩请求失败", "page", p, "err", err)
				mu.Unlock()
				return
			}
			postResp, err := session.HttpClient.Do(postReq)
			if err != nil {
				mu.Lock()
				failed[p-1] = true
				slog.Warn("获取成绩请求失败", "page", p, "err", err)
				mu.Unlock()
				return
			}
			pb, err := io.ReadAll(postResp.Body)
			postResp.Body.Close()
			if err != nil {
				mu.Lock()
				failed[p-1] = true
				slog.Warn("读取成绩响应失败", "page", p, "err", err)
				mu.Unlock()
				return
			}
			var pr scorePageResp
			if err := json.Unmarshal(pb, &pr); err != nil {
				mu.Lock()
				failed[p-1] = true
				slog.Warn("解析成绩响应失败", "page", p, "err", err)
				mu.Unlock()
				return
			}
			mu.Lock()
			results[p-1] = pr
			mu.Unlock()
		}(page)
	}
	wg.Wait()

	// 合并所有页（跳过失败页）
	var allScores []model.Score
	for i, resp := range results {
		if failed[i] {
			continue
		}
		allScores = append(allScores, parseScoreRows(resp.Rows, &seen)...)
	}
	if len(allScores) < totalRows {
		slog.Warn("成绩分页不完整，部分页抓取失败", "collected", len(allScores), "expected", totalRows)
	}
	return allScores, nil
}

func parseScoreRows(rows []struct {
	Xn string `json:"xn"`; Xq string `json:"xq"`; Ssbjmc string `json:"ssbjmc"`
	Kcmc string `json:"kcmc"`; Kcxz string `json:"kcxz"`; Kcxf string `json:"kcfxf"`
	Zdjsxm string `json:"zdjsxm"`; Cj any `json:"cj"`; Cjjd string `json:"cjjd"`
	Cjsx string `json:"cjsx"`
}, seen *sync.Map) []model.Score {
	var out []model.Score
	for _, r := range rows {
		key := r.Xn + r.Xq + r.Kcmc
		if _, exists := seen.Load(key); !exists {
			seen.Store(key, struct{}{})
			credit, _ := strconv.ParseFloat(r.Kcxf, 64)
			gpa, _ := strconv.ParseFloat(r.Cjjd, 64)
			out = append(out, model.Score{
				Year: r.Xn, Semester: r.Xq, ClassName: r.Ssbjmc,
				Course: r.Kcmc, Nature: r.Kcxz, Credit: credit,
				Teacher: r.Zdjsxm, Grade: fmt.Sprintf("%v", r.Cj),
				GPA: gpa, Type: r.Cjsx,
			})
		}
	}
	return out
}

type scorePageResp struct {
	Total string `json:"total"` // 教务系统返回的是字符串 "40"
	Rows  []struct {
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
	} `json:"rows"`
}

// jar cookie jar 实现
type jar struct {
	cookies map[string][]*http.Cookie
	mu      sync.Mutex
}

func (j *jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.cookies == nil {
		j.cookies = make(map[string][]*http.Cookie)
	}
	// B2 修复：合并而非覆盖，以新 cookie name 为准，不存在则追加
	existing := j.cookies[u.Host]
	cookieMap := make(map[string]*http.Cookie)
	for _, c := range existing {
		cookieMap[c.Name] = c
	}
	for _, c := range cookies {
		cookieMap[c.Name] = c
	}
	merged := make([]*http.Cookie, 0, len(cookieMap))
	for _, c := range cookieMap {
		merged = append(merged, c)
	}
	j.cookies[u.Host] = merged
}

func (j *jar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.cookies[u.Host]
}
