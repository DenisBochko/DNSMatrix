package mailer

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
)

type Mailer interface {
	SendHTML(to, subject, htmlTpl string, data any) error
}

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string // "Name <no-reply@hackathon.local>" или просто "no-reply@hackathon.local"
	UseTLS   bool   // true = SMTPS (465) или явный TLS-туннель; false = обычный TCP без AUTH
}

type mailer struct {
	cfg *Config
}

func New(cfg *Config) Mailer {
	return &mailer{cfg: cfg}
}

func (m *mailer) SendHTML(to, subject, htmlTpl string, data any) error {
	t, err := template.New("email").Parse(htmlTpl)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var body bytes.Buffer
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("exec template: %w", err)
	}

	msg := buildMessage(m.cfg.From, to, subject, body.String())
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	from := parseFromEmail(m.cfg.From)

	// AUTH используем ТОЛЬКО если TLS включен и заданы креды
	var auth smtp.Auth
	if m.cfg.UseTLS && m.cfg.Username != "" && m.cfg.Password != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	if m.cfg.UseTLS {
		return sendTLS(addr, m.cfg.Host, auth, from, to, msg)
	}

	// Без TLS — никогда не передаем auth, иначе PlainAuth взорвется "unencrypted connection"
	return smtp.SendMail(addr, nil, from, []string{to}, []byte(msg))
}

func buildMessage(from, to, subject, htmlBody string) string {
	headers := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + htmlBody
}

func parseFromEmail(from string) string {
	if i := strings.Index(from, "<"); i >= 0 {
		if j := strings.Index(from[i:], ">"); j > 0 {
			return strings.TrimSpace(from[i+1 : i+j])
		}
	}
	return strings.TrimSpace(from)
}

func sendTLS(addr, host string, auth smtp.Auth, from string, to string, msg string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("dial tls: %w", err)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}
	defer func() {
		_ = c.Close()
	}()

	// AUTH только если задан
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}

	if err := c.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return w.Close()
}
