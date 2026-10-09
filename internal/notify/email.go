package notify

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"jww/internal/model"
)

// SMTPConfig 发信邮箱
type SMTPConfig struct {
	Host     string
	Port     int // 465 使用 SSL，其它端口（如 587）使用 STARTTLS
	User     string
	Password string
	From     string // 默认同 User
}

func buildEmail(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + (&mail.Address{Name: "课表监控", Address: from}).String() + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + mime.BEncoding.Encode("UTF-8", subject) + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n")
	enc := base64.StdEncoding.EncodeToString([]byte(body))
	for len(enc) > 76 {
		b.WriteString(enc[:76] + "\r\n")
		enc = enc[76:]
	}
	b.WriteString(enc + "\r\n")
	return []byte(b.String())
}

func (n *Notifier) sendEmail(c *model.EmailChannel, p *Payload) error {
	cfg := n.smtp
	if cfg == nil {
		return errors.New("服务器未配置发信邮箱")
	}
	msg := buildEmail(cfg.From, c.To, Title(p.Event), p.Text)
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
	if cfg.Port != 465 {
		return smtp.SendMail(addr, auth, cfg.From, []string{c.To}, msg)
	}

	// 465：直接 TLS 连接（QQ/163 邮箱常用）
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, &tls.Config{ServerName: cfg.Host})
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Close()
	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(c.To); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}
