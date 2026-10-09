package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"jww/pkg/database"
)

// 用户可同时开启多个通知渠道：微信（PushPlus）、邮箱、群机器人（企业微信/钉钉/飞书），以及原有的 Webhook。

// NotifyChannels 用户的通知渠道配置（nil 表示未开启）
type NotifyChannels struct {
	PushPlus *PushPlusChannel `json:"pushplus"`
	Email    *EmailChannel    `json:"email"`
	Bot      *BotChannel      `json:"bot"`
}

// PushPlusChannel 微信推送（https://www.pushplus.plus 扫码获取 token）
type PushPlusChannel struct {
	Token string `json:"token"`
}

// EmailChannel 邮件推送（需要服务器配置 SMTP）
type EmailChannel struct {
	To string `json:"to"`
}

// BotChannel 群机器人
type BotChannel struct {
	Type   string `json:"type"`   // wecom | dingtalk | feishu
	URL    string `json:"url"`    // 机器人 Webhook 地址
	Secret string `json:"secret"` // 钉钉/飞书“加签”密钥（可选）
}

const (
	BotWeCom    = "wecom"
	BotDingTalk = "dingtalk"
	BotFeishu   = "feishu"
)

// 各类机器人允许的域名（同时防止被用来请求内网地址）
var botHosts = map[string][]string{
	BotWeCom:    {"qyapi.weixin.qq.com"},
	BotDingTalk: {"oapi.dingtalk.com"},
	BotFeishu:   {"open.feishu.cn", "open.larksuite.com"},
}

// Validate 校验并规范化配置
func (c *NotifyChannels) Validate() error {
	if c.PushPlus != nil {
		c.PushPlus.Token = strings.TrimSpace(c.PushPlus.Token)
		if c.PushPlus.Token == "" {
			return errors.New("请填写 PushPlus token")
		}
	}
	if c.Email != nil {
		c.Email.To = strings.TrimSpace(c.Email.To)
		addr, err := mail.ParseAddress(c.Email.To)
		if err != nil || addr.Address != c.Email.To {
			return errors.New("邮箱地址格式不正确")
		}
		if !EmailAvailable() {
			return errors.New("服务器未配置发信邮箱，暂不能使用邮件通知")
		}
	}
	if c.Bot != nil {
		c.Bot.URL = strings.TrimSpace(c.Bot.URL)
		c.Bot.Secret = strings.TrimSpace(c.Bot.Secret)
		hosts, ok := botHosts[c.Bot.Type]
		if !ok {
			return errors.New("不支持的机器人类型")
		}
		u, err := url.Parse(c.Bot.URL)
		if err != nil || u.Scheme != "https" || !containsString(hosts, u.Hostname()) {
			return fmt.Errorf("机器人地址应以 https://%s 开头", hosts[0])
		}
	}
	return nil
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Any 是否至少开启了一个渠道
func (c *NotifyChannels) Any() bool {
	return c != nil && (c.PushPlus != nil || c.Email != nil || c.Bot != nil)
}

const notifyChannelsPrefix = "notify:channels:"

// SaveNotifyChannels 保存用户的通知渠道
func SaveNotifyChannels(uid string, c *NotifyChannels) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	ctx, cancel := storeCtx()
	defer cancel()
	return database.GetRedis().Set(ctx, notifyChannelsPrefix+uid, data, 0).Err()
}

// GetNotifyChannels 读取用户的通知渠道，未配置时返回空配置
func GetNotifyChannels(uid string) (*NotifyChannels, error) {
	ctx, cancel := storeCtx()
	defer cancel()
	data, err := database.GetRedis().Get(ctx, notifyChannelsPrefix+uid).Bytes()
	if errors.Is(err, redis.Nil) {
		return &NotifyChannels{}, nil
	}
	if err != nil {
		return nil, err
	}
	var c NotifyChannels
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func notifyTitle(event string) string {
	switch event {
	case EventScheduleDiff:
		return "课表变动提醒"
	case EventSessionExpired:
		return "教务登录已失效"
	case EventTest:
		return "课表监控测试通知"
	}
	return "课表监控通知"
}

// ---- 发送 ----

// postJSON 发送 JSON 并把响应解析到 out
func postJSON(rawURL string, body interface{}, out interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := notifyClient.Post(rawURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("响应解析失败: %s", truncate(string(respBody), 200))
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

var pushPlusURL = "https://www.pushplus.plus/send"

func sendPushPlus(ch *PushPlusChannel, p *NotifyPayload) error {
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	err := postJSON(pushPlusURL, map[string]string{
		"token":    ch.Token,
		"title":    notifyTitle(p.Event),
		"content":  p.Text,
		"template": "txt",
	}, &resp)
	if err != nil {
		return err
	}
	if resp.Code != 200 {
		return fmt.Errorf("PushPlus: %s", resp.Msg)
	}
	return nil
}

// signBot 钉钉/飞书加签
func signBot(key, msg string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(msg))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func sendBot(ch *BotChannel, p *NotifyPayload) error {
	text := "【" + notifyTitle(p.Event) + "】\n" + p.Text
	ts := time.Now()

	switch ch.Type {
	case BotWeCom:
		var resp struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		if err := postJSON(ch.URL, map[string]interface{}{
			"msgtype": "text",
			"text":    map[string]string{"content": text},
		}, &resp); err != nil {
			return err
		}
		if resp.ErrCode != 0 {
			return fmt.Errorf("企业微信: %s", resp.ErrMsg)
		}

	case BotDingTalk:
		target := ch.URL
		if ch.Secret != "" {
			u, err := url.Parse(ch.URL)
			if err != nil {
				return err
			}
			ms := strconv.FormatInt(ts.UnixMilli(), 10)
			q := u.Query()
			q.Set("timestamp", ms)
			q.Set("sign", signBot(ch.Secret, ms+"\n"+ch.Secret))
			u.RawQuery = q.Encode()
			target = u.String()
		}
		var resp struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		if err := postJSON(target, map[string]interface{}{
			"msgtype": "text",
			"text":    map[string]string{"content": text},
		}, &resp); err != nil {
			return err
		}
		if resp.ErrCode != 0 {
			return fmt.Errorf("钉钉: %s", resp.ErrMsg)
		}

	case BotFeishu:
		body := map[string]interface{}{
			"msg_type": "text",
			"content":  map[string]string{"text": text},
		}
		if ch.Secret != "" {
			sec := strconv.FormatInt(ts.Unix(), 10)
			body["timestamp"] = sec
			body["sign"] = signBot(sec+"\n"+ch.Secret, "")
		}
		var resp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if err := postJSON(ch.URL, body, &resp); err != nil {
			return err
		}
		if resp.Code != 0 {
			return fmt.Errorf("飞书: %s", resp.Msg)
		}

	default:
		return fmt.Errorf("不支持的机器人类型 %q", ch.Type)
	}
	return nil
}

// ---- 邮件 ----

// SMTPConfig 发信邮箱配置
type SMTPConfig struct {
	Host     string
	Port     int // 465 使用 SSL，其它端口使用 STARTTLS（如 587）
	User     string
	Password string
	From     string // 发件人地址，默认同 User
}

var smtpConfig *SMTPConfig

// ConfigureEmail 设置发信邮箱；Host 为空时邮件通知不可用
func ConfigureEmail(cfg SMTPConfig) {
	if cfg.Host == "" || cfg.User == "" {
		smtpConfig = nil
		return
	}
	if cfg.From == "" {
		cfg.From = cfg.User
	}
	smtpConfig = &cfg
}

// EmailAvailable 服务器是否配置了发信邮箱
func EmailAvailable() bool {
	return smtpConfig != nil
}

func buildEmail(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + (&mail.Address{Name: "课表监控", Address: from}).String() + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + mime.BEncoding.Encode("UTF-8", subject) + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	enc := base64.StdEncoding.EncodeToString([]byte(body))
	for len(enc) > 76 {
		b.WriteString(enc[:76] + "\r\n")
		enc = enc[76:]
	}
	b.WriteString(enc + "\r\n")
	return []byte(b.String())
}

func sendEmail(ch *EmailChannel, p *NotifyPayload) error {
	cfg := smtpConfig
	if cfg == nil {
		return errors.New("服务器未配置发信邮箱")
	}
	msg := buildEmail(cfg.From, ch.To, notifyTitle(p.Event), p.Text)
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)

	if cfg.Port != 465 {
		return smtp.SendMail(addr, auth, cfg.From, []string{ch.To}, msg)
	}

	// 465 端口：直接 TLS 连接（QQ/163 邮箱常用）
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, &tls.Config{ServerName: cfg.Host})
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if err := c.Auth(auth); err != nil {
		return err
	}
	if err := c.Mail(cfg.From); err != nil {
		return err
	}
	if err := c.Rcpt(ch.To); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
