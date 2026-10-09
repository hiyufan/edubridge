// Package jwclient 封装与学校教务系统（学生门户）的 HTTP 交互和页面解析。
//
// 一个 Client 对应一个教务登录会话（持有 cookie），只负责收发请求、解析页面，
// 不做缓存、不做持久化，也不计算“当前周”等业务含义。
package jwclient

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL 学校教务系统地址
const DefaultBaseURL = "https://jw.fzrjxy.com"

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

var (
	// ErrSessionExpired 教务登录态已失效，需要用户重新输入验证码登录
	ErrSessionExpired = errors.New("教务系统登录已失效，请重新登录")
	// ErrUnavailable 教务系统网络错误或返回异常
	ErrUnavailable = errors.New("教务系统暂时无法访问，请稍后再试")
)

// LoginError 教务系统拒绝登录（验证码错误、密码错误等），Info 可直接展示给用户
type LoginError struct {
	Info string
}

func (e *LoginError) Error() string { return e.Info }

// Client 一个教务登录会话
type Client struct {
	baseURL string
	http    *http.Client
	jar     *Jar
}

// New 创建一个新会话（尚未登录）
func New(baseURL string) *Client {
	jar := &Jar{}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		jar:     jar,
		http: &http.Client{
			Timeout: 20 * time.Second,
			Jar:     jar,
			Transport: &http.Transport{
				Proxy:               http.ProxyFromEnvironment,
				MaxIdleConns:        10,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 20 * time.Second,
			},
		},
	}
}

// Jar 会话 cookie（用于持久化与恢复）
func (c *Client) Jar() *Jar { return c.jar }

func (c *Client) newRequest(ctx context.Context, method, target string, body io.Reader) (*http.Request, error) {
	if strings.HasPrefix(target, "/") {
		target = c.baseURL + target
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/json,*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Referer", c.baseURL+"/")
	return req, nil
}

type page struct {
	body     []byte
	finalURL string
	status   int
	header   http.Header
}

func (c *Client) do(req *http.Request) (*page, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return &page{body: body, finalURL: resp.Request.URL.String(), status: resp.StatusCode, header: resp.Header}, nil
}

func (c *Client) get(ctx context.Context, target string) (*page, error) {
	req, err := c.newRequest(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) postForm(ctx context.Context, target string, form url.Values) (*page, error) {
	req, err := c.newRequest(ctx, http.MethodPost, target, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.do(req)
}

// Captcha 获取登录验证码图片
func (c *Client) Captcha(ctx context.Context) (image []byte, contentType string, err error) {
	p, err := c.get(ctx, "/studentportal.php/Public/verify/")
	if err != nil {
		return nil, "", err
	}
	if p.status != http.StatusOK || len(p.body) == 0 {
		return nil, "", fmt.Errorf("%w: 验证码 HTTP %d", ErrUnavailable, p.status)
	}
	contentType = p.header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}
	return p.body, contentType, nil
}

// Login 用学号/证件号 + 密码 + 验证码登录（需先在同一 Client 上获取验证码）
func (c *Client) Login(ctx context.Context, username, password, captcha, loginType string) error {
	field := "xsxh"
	switch loginType {
	case "zjh", "gkksh":
		field = loginType
	}
	hash := md5.Sum([]byte(password))

	form := url.Values{}
	form.Set("logintype", loginType)
	form.Set(field, username)
	form.Set("dlmm", hex.EncodeToString(hash[:]))
	form.Set("yzm", captcha)
	p, err := c.postForm(ctx, "/studentportal.php/Index/checkLogin", form)
	if err != nil {
		return err
	}

	var result struct {
		Status  int    `json:"status"`
		Info    string `json:"info"`
		Gotourl string `json:"gotourl"`
	}
	if err := json.Unmarshal(p.body, &result); err != nil {
		return fmt.Errorf("%w: 登录响应无法解析", ErrUnavailable)
	}
	if result.Status != 1 {
		return &LoginError{Info: result.Info}
	}

	// 访问跳转页建立完整会话；登录本身已成功，这一步失败不影响结果
	next := result.Gotourl
	if next == "" {
		next = "/studentportal.php/Main/"
	}
	_, _ = c.get(ctx, next)
	return nil
}

// Ping 访问学生门户首页：检测登录态，同时让学校那边的会话保持活跃
func (c *Client) Ping(ctx context.Context) error {
	p, err := c.get(ctx, "/studentportal.php/Main/")
	if err != nil {
		return err
	}
	if isLoginPage(p.finalURL, string(p.body)) {
		return ErrSessionExpired
	}
	return nil
}
