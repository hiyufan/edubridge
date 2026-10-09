package notify

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"jww/internal/model"
	"jww/internal/store"
)

var (
	ctx         = context.Background()
	testPayload = &Payload{Event: EventScheduleDiff, Text: "课表有变动：\n+ 大学物理"}
)

func newNotifier(t *testing.T, opts Options) (*Notifier, *store.Store) {
	t.Helper()
	mr := miniredis.RunT(t)
	st := store.New(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	return New(st, opts), st
}

type captured struct {
	r    *http.Request
	body []byte
}

// recorder 记录请求并按 reply 回复
func recorder(t *testing.T, reply string) (*httptest.Server, chan captured) {
	ch := make(chan captured, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		ch <- captured{r, b}
		io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv, ch
}

func decode(t *testing.T, b []byte) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestValidateChannels(t *testing.T) {
	n, _ := newNotifier(t, Options{})
	cases := []struct {
		name string
		c    model.NotifyChannels
		ok   bool
	}{
		{"empty", model.NotifyChannels{}, true},
		{"pushplus", model.NotifyChannels{PushPlus: &model.PushPlusChannel{Token: " abc "}}, true},
		{"pushplus blank", model.NotifyChannels{PushPlus: &model.PushPlusChannel{Token: " "}}, false},
		{"wecom", model.NotifyChannels{Bot: &model.BotChannel{Type: BotWeCom, URL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=x"}}, true},
		{"wecom wrong host", model.NotifyChannels{Bot: &model.BotChannel{Type: BotWeCom, URL: "https://127.0.0.1/send"}}, false},
		{"dingtalk http", model.NotifyChannels{Bot: &model.BotChannel{Type: BotDingTalk, URL: "http://oapi.dingtalk.com/robot/send"}}, false},
		{"feishu", model.NotifyChannels{Bot: &model.BotChannel{Type: BotFeishu, URL: "https://open.feishu.cn/open-apis/bot/v2/hook/x"}}, true},
		{"unknown bot", model.NotifyChannels{Bot: &model.BotChannel{Type: "slack", URL: "https://hooks.slack.com/x"}}, false},
		{"email without smtp", model.NotifyChannels{Email: &model.EmailChannel{To: "a@example.com"}}, false},
	}
	for _, tc := range cases {
		if err := n.ValidateChannels(&tc.c); (err == nil) != tc.ok {
			t.Errorf("%s: %v, want ok=%v", tc.name, err, tc.ok)
		}
	}

	withSMTP, _ := newNotifier(t, Options{SMTP: SMTPConfig{Host: "smtp.example.com", Port: 465, User: "bot@example.com"}})
	if err := withSMTP.ValidateChannels(&model.NotifyChannels{Email: &model.EmailChannel{To: "a@example.com"}}); err != nil {
		t.Errorf("email with smtp: %v", err)
	}
	if err := withSMTP.ValidateChannels(&model.NotifyChannels{Email: &model.EmailChannel{To: "张三 <a@example.com>"}}); err == nil {
		t.Error("display name should be rejected")
	}
}

func TestWebhookSignature(t *testing.T) {
	srv, got := recorder(t, "")
	n, _ := newNotifier(t, Options{})
	if err := n.SendWebhook(ctx, &model.WebhookConfig{URL: srv.URL, Secret: "s3"}, testPayload); err != nil {
		t.Fatal(err)
	}
	c := <-got
	mac := hmac.New(sha256.New, []byte("s3"))
	mac.Write(c.body)
	if c.r.Header.Get("X-Hub-Signature-256") != "sha256="+hex.EncodeToString(mac.Sum(nil)) || c.r.Header.Get("X-JWW-Event") != EventScheduleDiff {
		t.Fatalf("headers = %v", c.r.Header)
	}
	if m := decode(t, c.body); m["event"] != EventScheduleDiff || m["text"] != testPayload.Text {
		t.Fatalf("body = %v", m)
	}
}

func TestPushPlus(t *testing.T) {
	srv, got := recorder(t, `{"code":200,"msg":"ok"}`)
	n, _ := newNotifier(t, Options{PushPlusURL: srv.URL})
	if err := n.sendPushPlus(ctx, &model.PushPlusChannel{Token: "tk"}, testPayload); err != nil {
		t.Fatal(err)
	}
	if m := decode(t, (<-got).body); m["token"] != "tk" || m["title"] != "课表变动提醒" || m["content"] != testPayload.Text {
		t.Fatalf("body = %v", m)
	}

	bad, _ := recorder(t, `{"code":903,"msg":"无效的用户token"}`)
	n2, _ := newNotifier(t, Options{PushPlusURL: bad.URL})
	if err := n2.sendPushPlus(ctx, &model.PushPlusChannel{Token: "x"}, testPayload); err == nil || !strings.Contains(err.Error(), "无效的用户token") {
		t.Fatalf("err = %v", err)
	}
}

func TestBots(t *testing.T) {
	n, _ := newNotifier(t, Options{})

	srv, got := recorder(t, `{"errcode":0}`)
	if err := n.sendBot(ctx, &model.BotChannel{Type: BotWeCom, URL: srv.URL}, testPayload); err != nil {
		t.Fatal(err)
	}
	if m := decode(t, (<-got).body); !strings.HasPrefix(m["text"].(map[string]any)["content"].(string), "【课表变动提醒】") {
		t.Fatalf("wecom body = %v", m)
	}

	srv, got = recorder(t, `{"errcode":0}`)
	if err := n.sendBot(ctx, &model.BotChannel{Type: BotDingTalk, URL: srv.URL + "/robot/send?access_token=a", Secret: "SEC"}, testPayload); err != nil {
		t.Fatal(err)
	}
	q := (<-got).r.URL.Query()
	if q.Get("access_token") != "a" || q.Get("sign") != signBot("SEC", q.Get("timestamp")+"\nSEC") {
		t.Fatalf("dingtalk query = %v", q)
	}

	srv, _ = recorder(t, `{"errcode":310000,"errmsg":"keywords not in content"}`)
	if err := n.sendBot(ctx, &model.BotChannel{Type: BotDingTalk, URL: srv.URL}, testPayload); err == nil || !strings.Contains(err.Error(), "keywords") {
		t.Fatalf("dingtalk error = %v", err)
	}

	srv, got = recorder(t, `{"code":0}`)
	if err := n.sendBot(ctx, &model.BotChannel{Type: BotFeishu, URL: srv.URL, Secret: "SEC"}, testPayload); err != nil {
		t.Fatal(err)
	}
	m := decode(t, (<-got).body)
	if ts, _ := m["timestamp"].(string); m["sign"] != signBot(ts+"\nSEC", "") || m["msg_type"] != "text" {
		t.Fatalf("feishu body = %v", m)
	}
}

// fakeSMTP 极简 SMTP 服务器（无 TLS，仅本机测试）
func fakeSMTP(t *testing.T) (string, chan string) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	got := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		write := func(s string) { io.WriteString(conn, s+"\r\n") }
		write("220 fake")
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
			case strings.HasPrefix(cmd, "EHLO"):
				write("250-fake")
				write("250 AUTH PLAIN")
			case strings.HasPrefix(cmd, "AUTH"):
				write("235 ok")
			case strings.HasPrefix(cmd, "DATA"):
				write("354 go")
				var b strings.Builder
				for {
					l, err := r.ReadString('\n')
					if err != nil || l == ".\r\n" {
						break
					}
					b.WriteString(l)
				}
				got <- b.String()
				write("250 queued")
			case strings.HasPrefix(cmd, "QUIT"):
				write("221 bye")
				return
			default:
				write("250 ok")
			}
		}
	}()
	return ln.Addr().String(), got
}

func TestEmail(t *testing.T) {
	addr, got := fakeSMTP(t)
	host, port, _ := net.SplitHostPort(addr)
	p, _ := strconv.Atoi(port)
	n, _ := newNotifier(t, Options{SMTP: SMTPConfig{Host: host, Port: p, User: "bot@example.com", Password: "pw"}})

	if err := n.sendEmail(&model.EmailChannel{To: "me@example.com"}, testPayload); err != nil {
		t.Fatal(err)
	}
	var raw string
	select {
	case raw = <-got:
	case <-time.After(3 * time.Second):
		t.Fatal("no mail")
	}
	msg, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	subject, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	body, _ := io.ReadAll(msg.Body)
	text, _ := base64.StdEncoding.DecodeString(strings.ReplaceAll(string(body), "\r\n", ""))
	if subject != "课表变动提醒" || msg.Header.Get("To") != "me@example.com" || string(text) != testPayload.Text {
		t.Fatalf("subject=%q to=%q text=%q", subject, msg.Header.Get("To"), text)
	}
}

func TestSendFansOutAndRecordsHistory(t *testing.T) {
	pp, ppGot := recorder(t, `{"code":200}`)
	bot, _ := recorder(t, `{"errcode":93000,"errmsg":"invalid webhook url"}`)
	n, st := newNotifier(t, Options{PushPlusURL: pp.URL})
	const uid = "u"

	if _, err := n.Send(ctx, uid, testPayload); !errors.Is(err, ErrNoChannel) {
		t.Fatalf("no channel: %v", err)
	}

	st.SaveChannels(ctx, uid, &model.NotifyChannels{
		PushPlus: &model.PushPlusChannel{Token: "tk"},
		Bot:      &model.BotChannel{Type: BotWeCom, URL: bot.URL},
	})
	results, err := n.Send(ctx, uid, testPayload)
	if err == nil || !strings.Contains(err.Error(), "invalid webhook url") {
		t.Fatalf("err = %v", err)
	}
	<-ppGot
	if len(results) != 2 || results[0].Error != "" || results[1].Error == "" {
		t.Fatalf("results = %+v", results)
	}

	n.Notify(uid, EventScoreNew, "出成绩了", nil)
	n.Notify(uid, EventReminder, "上课提醒不进动态", nil)
	n.Wait()
	if h, _ := st.History(ctx, uid, 10); len(h) != 1 || h[0].Event != EventScoreNew {
		t.Fatalf("history = %+v", h)
	}
}
