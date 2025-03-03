package nsa

import (
	"fmt"
	"log/slog"
	"net/smtp"
	"os"
)

type Mail struct {
	subject string
	id      string
	title   string
}

func NewMail() *Mail {
	return &Mail{}
}

func (m *Mail) Subject(s string) *Mail {
	m.subject = s
	return m
}

func (m *Mail) Id(id string) *Mail {
	m.id = id
	return m
}

func (m *Mail) Title(t string) *Mail {
	m.title = t
	return m
}

func (m *Mail) Send() error {
	msg := "title: " + m.title + "\r\n" + "URL: " + fmt.Sprintf("https://www.youtube.com/watch?v=%s", m.id)
	addr := os.Getenv("MAIL_ADDRESS")
	auth := smtp.PlainAuth(
		"",
		addr,                       // 送信に使うアカウント
		os.Getenv("SMTP_PASSWORD"), // アカウントのパスワード or アプリケーションパスワード
		"smtp.gmail.com",
	)

	slog.Info(
		"send-mail",
		slog.String("severity", "INFO"),
		slog.String("subject", m.subject),
		slog.String("message", msg),
	)

	return smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		addr,           // 送信元
		[]string{addr}, // 送信先
		[]byte(
			"To: "+addr+"\r\n"+
				"Subject:"+m.subject+"\r\n"+
				"\r\n"+
				msg+"\r\n"),
	)
}
