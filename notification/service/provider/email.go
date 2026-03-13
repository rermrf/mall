package provider

import (
	"context"
	"fmt"
	"net/smtp"
)

type EmailProvider interface {
	Send(ctx context.Context, to string, subject string, body string) error
}

type SmtpEmailProvider struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func NewSmtpEmailProvider(host string, port int, username, password, from string) EmailProvider {
	return &SmtpEmailProvider{
		host: host, port: port, username: username, password: password, from: from,
	}
}

func (p *SmtpEmailProvider) Send(ctx context.Context, to string, subject string, body string) error {
	addr := fmt.Sprintf("%s:%d", p.host, p.port)
	auth := smtp.PlainAuth("", p.username, p.password, p.host)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		p.from, to, subject, body)
	err := smtp.SendMail(addr, auth, p.from, []string{to}, []byte(msg))
	if err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}
	return nil
}
