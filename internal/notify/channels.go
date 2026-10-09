package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"jww/internal/model"
)

// 群机器人类型
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

// ValidateChannels 校验并规范化用户提交的渠道配置
func (n *Notifier) ValidateChannels(c *model.NotifyChannels) error {
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
		if !n.EmailAvailable() {
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
		if err != nil || u.Scheme != "https" || !slices.Contains(hosts, u.Hostname()) {
			return fmt.Errorf("机器人地址应以 https://%s 开头", hosts[0])
		}
	}
	return nil
}

// postJSON 发送 JSON 并把响应解析到 out
func (n *Notifier) postJSON(ctx context.Context, target string, body, out any, header http.Header) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header[k] = v
	}
	resp, err := n.client.Do(req)
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
			return fmt.Errorf("响应无法解析: %.200s", respBody)
		}
	}
	return nil
}

// SendWebhook 推送到用户自定义 Webhook；配置了密钥时附带 HMAC-SHA256 签名
func (n *Notifier) SendWebhook(ctx context.Context, w *model.WebhookConfig, p *Payload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	header := http.Header{"X-Jww-Event": {p.Event}}
	if w.Secret != "" {
		mac := hmac.New(sha256.New, []byte(w.Secret))
		mac.Write(body)
		header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	return n.postJSON(ctx, w.URL, json.RawMessage(body), nil, header)
}

func (n *Notifier) sendPushPlus(ctx context.Context, c *model.PushPlusChannel, p *Payload) error {
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	err := n.postJSON(ctx, n.pushPlusURL, map[string]string{
		"token": c.Token, "title": Title(p.Event), "content": p.Text, "template": "txt",
	}, &resp, nil)
	if err != nil {
		return err
	}
	if resp.Code != 200 {
		return fmt.Errorf("PushPlus: %s", resp.Msg)
	}
	return nil
}

// signBot 钉钉/飞书加签：HMAC-SHA256 后 base64
func signBot(key, msg string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(msg))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (n *Notifier) sendBot(ctx context.Context, c *model.BotChannel, p *Payload) error {
	text := "【" + Title(p.Event) + "】\n" + p.Text
	textMsg := map[string]any{"msgtype": "text", "text": map[string]string{"content": text}}
	var resp struct {
		ErrCode int    `json:"errcode"` // 企业微信、钉钉
		ErrMsg  string `json:"errmsg"`
		Code    int    `json:"code"` // 飞书
		Msg     string `json:"msg"`
	}

	switch c.Type {
	case BotWeCom:
		if err := n.postJSON(ctx, c.URL, textMsg, &resp, nil); err != nil {
			return err
		}
		if resp.ErrCode != 0 {
			return fmt.Errorf("企业微信: %s", resp.ErrMsg)
		}
	case BotDingTalk:
		target := c.URL
		if c.Secret != "" {
			u, err := url.Parse(c.URL)
			if err != nil {
				return err
			}
			ms := strconv.FormatInt(time.Now().UnixMilli(), 10)
			q := u.Query()
			q.Set("timestamp", ms)
			q.Set("sign", signBot(c.Secret, ms+"\n"+c.Secret))
			u.RawQuery = q.Encode()
			target = u.String()
		}
		if err := n.postJSON(ctx, target, textMsg, &resp, nil); err != nil {
			return err
		}
		if resp.ErrCode != 0 {
			return fmt.Errorf("钉钉: %s", resp.ErrMsg)
		}
	case BotFeishu:
		body := map[string]any{"msg_type": "text", "content": map[string]string{"text": text}}
		if c.Secret != "" {
			sec := strconv.FormatInt(time.Now().Unix(), 10)
			body["timestamp"] = sec
			body["sign"] = signBot(sec+"\n"+c.Secret, "")
		}
		if err := n.postJSON(ctx, c.URL, body, &resp, nil); err != nil {
			return err
		}
		if resp.Code != 0 {
			return fmt.Errorf("飞书: %s", resp.Msg)
		}
	default:
		return fmt.Errorf("不支持的机器人类型 %q", c.Type)
	}
	return nil
}
