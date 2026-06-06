package enrollment

import (
	"fmt"
	"net/smtp"
)

// Mailer sends OTP codes via plain SMTP (Mailpit in the dev stack).
type Mailer struct {
	host string // e.g. "mailpit:1025"
	from string // e.g. "no-reply@banking.local"
}

func NewMailer(host, from string) *Mailer {
	return &Mailer{host: host, from: from}
}

// Send dispatches a single-purpose OTP email. No auth, no TLS — Mailpit
// accepts anything in the dev stack.
func (m *Mailer) Send(to, code string) error {
	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: Your Banking MFA code\r\n"+
		"\r\n"+
		"Your one-time MFA enrollment code is: %s\r\n"+
		"\r\n"+
		"It is valid for 5 minutes.\r\n", m.from, to, code)
	return smtp.SendMail(m.host, nil, m.from, []string{to}, []byte(msg))
}
