package service

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
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
)

func TestValidateChannels(t *testing.T) {
	ConfigureEmail(SMTPConfig{})
	cases := []struct {
		name string
		c    NotifyChannels
		ok   bool
	}{
		{"empty", NotifyChannels{}, true},
		{"pushplus", NotifyChannels{PushPlus: &PushPlusChannel{Token: " abc "}}, true},
		{"pushplus blank", NotifyChannels{PushPlus: &PushPlusChannel{Token: " "}}, false},
		{"wecom", NotifyChannels{Bot: &BotChannel{Type: BotWeCom, URL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=x"}}, true},
		{"wecom wrong host", NotifyChannels{Bot: &BotChannel{Type: BotWeCom, URL: "https://127.0.0.1/cgi-bin/webhook/send"}}, false},
		{"dingtalk http", NotifyChannels{Bot: &BotChannel{Type: BotDingTalk, URL: "http://oapi.dingtalk.com/robot/send?access_token=x"}}, false},
		{"feishu", NotifyChannels{Bot: &BotChannel{Type: BotFeishu, URL: "https://open.feishu.cn/open-apis/bot/v2/hook/x"}}, true},
		{"unknown bot", NotifyChannels{Bot: &BotChannel{Type: "slack", URL: "https://hooks.slack.com/x"}}, false},
		{"email without smtp", NotifyChannels{Email: &EmailChannel{To: "a@example.com"}}, false},
	}
	for _, tc := range cases {
		err := tc.c.Validate()
		if (err == nil) != tc.ok {
			t.Errorf("%s: Validate() = %v, want ok=%v", tc.name, err, tc.ok)
		}
	}

	ConfigureEmail(SMTPConfig{Host: "smtp.example.com", Port: 465, User: "bot@example.com"})
	defer ConfigureEmail(SMTPConfig{})
	if err := (&NotifyChannels{Email: &EmailChannel{To: "a@example.com"}}).Validate(); err != nil {
		t.Errorf("email with smtp: %v", err)
	}
	if err := (&NotifyChannels{Email: &EmailChannel{To: "张三 <a@example.com>"}}).Validate(); err == nil {
		t.Error("email with display name should be rejected")
	}
}

// recordJSON 返回一个记录请求的服务器，并按 reply 回复
func recordJSON(t *testing.T, reply string) (*httptest.Server, chan *http.Request, chan map[string]interface{}) {
	reqs := make(chan *http.Request, 1)
	bodies := make(chan map[string]interface{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		reqs <- r
		bodies <- body
		io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv, reqs, bodies
}

var testPayload = &NotifyPayload{Event: EventScheduleDiff, Text: "课表有变动：\n+ 大学物理"}

func TestSendPushPlus(t *testing.T) {
	srv, _, bodies := recordJSON(t, `{"code":200,"msg":"ok"}`)
	old := pushPlusURL
	pushPlusURL = srv.URL
	defer func() { pushPlusURL = old }()

	if err := sendPushPlus(&PushPlusChannel{Token: "tk"}, testPayload); err != nil {
		t.Fatal(err)
	}
	b := <-bodies
	if b["token"] != "tk" || b["title"] != "课表变动提醒" || b["content"] != testPayload.Text {
		t.Fatalf("body = %v", b)
	}

	srv2, _, _ := recordJSON(t, `{"code":903,"msg":"无效的用户token"}`)
	pushPlusURL = srv2.URL
	if err := sendPushPlus(&PushPlusChannel{Token: "bad"}, testPayload); err == nil || !strings.Contains(err.Error(), "无效的用户token") {
		t.Fatalf("expected token error, got %v", err)
	}
}

func TestSendBots(t *testing.T) {
	t.Run("wecom", func(t *testing.T) {
		srv, _, bodies := recordJSON(t, `{"errcode":0,"errmsg":"ok"}`)
		if err := sendBot(&BotChannel{Type: BotWeCom, URL: srv.URL + "/send?key=k"}, testPayload); err != nil {
			t.Fatal(err)
		}
		b := <-bodies
		if b["msgtype"] != "text" || !strings.Contains(b["text"].(map[string]interface{})["content"].(string), "【课表变动提醒】") {
			t.Fatalf("body = %v", b)
		}
	})

	t.Run("dingtalk signed", func(t *testing.T) {
		srv, reqs, _ := recordJSON(t, `{"errcode":0,"errmsg":"ok"}`)
		if err := sendBot(&BotChannel{Type: BotDingTalk, URL: srv.URL + "/robot/send?access_token=a", Secret: "SEC"}, testPayload); err != nil {
			t.Fatal(err)
		}
		q := (<-reqs).URL.Query()
		ts := q.Get("timestamp")
		if q.Get("access_token") != "a" || ts == "" {
			t.Fatalf("query = %v", q)
		}
		if want := signBot("SEC", ts+"\nSEC"); q.Get("sign") != want {
			t.Fatalf("sign = %q, want %q", q.Get("sign"), want)
		}
	})

	t.Run("dingtalk error", func(t *testing.T) {
		srv, _, _ := recordJSON(t, `{"errcode":310000,"errmsg":"keywords not in content"}`)
		err := sendBot(&BotChannel{Type: BotDingTalk, URL: srv.URL + "/robot/send?access_token=a"}, testPayload)
		if err == nil || !strings.Contains(err.Error(), "keywords") {
			t.Fatalf("expected error, got %v", err)
		}
	})

	t.Run("feishu signed", func(t *testing.T) {
		srv, _, bodies := recordJSON(t, `{"code":0,"msg":"success"}`)
		if err := sendBot(&BotChannel{Type: BotFeishu, URL: srv.URL + "/hook/x", Secret: "SEC"}, testPayload); err != nil {
			t.Fatal(err)
		}
		b := <-bodies
		ts, _ := b["timestamp"].(string)
		if _, err := strconv.ParseInt(ts, 10, 64); err != nil {
			t.Fatalf("timestamp = %v", b["timestamp"])
		}
		if want := signBot(ts+"\nSEC", ""); b["sign"] != want {
			t.Fatalf("sign = %v, want %q", b["sign"], want)
		}
		if b["msg_type"] != "text" {
			t.Fatalf("body = %v", b)
		}
	})
}

// fakeSMTP 极简 SMTP 服务器（无 TLS，仅用于本机测试），返回收到的 DATA
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
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
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

func TestSendEmail(t *testing.T) {
	addr, got := fakeSMTP(t)
	host, port, _ := net.SplitHostPort(addr)
	p, _ := strconv.Atoi(port)
	ConfigureEmail(SMTPConfig{Host: host, Port: p, User: "bot@example.com", Password: "pw"})
	defer ConfigureEmail(SMTPConfig{})

	if err := sendEmail(&EmailChannel{To: "me@example.com"}, testPayload); err != nil {
		t.Fatal(err)
	}

	var raw string
	select {
	case raw = <-got:
	case <-time.After(3 * time.Second):
		t.Fatal("no mail received")
	}
	msg, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	subject, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if subject != "课表变动提醒" || msg.Header.Get("To") != "me@example.com" {
		t.Fatalf("header = %v (subject %q)", msg.Header, subject)
	}
	body, _ := io.ReadAll(msg.Body)
	text, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(string(body), "\r\n", ""))
	if err != nil || string(text) != testPayload.Text {
		t.Fatalf("body = %q, err = %v", text, err)
	}
}

func TestSendNotifyFansOut(t *testing.T) {
	setupRedis(t)
	const uid = "2024009"

	if _, err := SendNotify(uid, testPayload); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	pp, _, ppBodies := recordJSON(t, `{"code":200,"msg":"ok"}`)
	old := pushPlusURL
	pushPlusURL = pp.URL
	defer func() { pushPlusURL = old }()
	bot, _, _ := recordJSON(t, `{"errcode":93000,"errmsg":"invalid webhook url"}`)

	SaveNotifyChannels(uid, &NotifyChannels{
		PushPlus: &PushPlusChannel{Token: "tk"},
		Bot:      &BotChannel{Type: BotWeCom, URL: bot.URL},
	})

	results, err := SendNotify(uid, testPayload)
	if err == nil || !strings.Contains(err.Error(), "invalid webhook url") {
		t.Fatalf("expected bot error, got %v", err)
	}
	<-ppBodies // 机器人失败不影响微信推送
	if len(results) != 2 || results[0].Channel != "pushplus" || results[0].Error != "" ||
		results[1].Channel != "bot" || results[1].Error == "" {
		t.Fatalf("results = %+v", results)
	}
}
