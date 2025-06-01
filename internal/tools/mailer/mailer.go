package mailer

import "gopkg.in/gomail.v2"

func SendEmail(from, to, subject, body, smtpHost string, smtpPort int, username, password string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	d := gomail.NewDialer(smtpHost, smtpPort, username, password)

	return d.DialAndSend(m)
}
