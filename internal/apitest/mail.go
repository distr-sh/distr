package apitest

import (
	"context"
	"slices"
	"sync"

	"github.com/go-mailx/mailx"
)

// Mail is a message the API sent through the instance mailer. Mail sent through an
// organization's own SMTP configuration does not pass through it.
type Mail struct {
	From     string
	To       []string
	Subject  string
	HTMLBody string
	TextBody string
}

// Mails returns what the API sent so far, oldest first.
func (h *Harness) Mails() []Mail {
	h.mails.mu.Lock()
	defer h.mails.mu.Unlock()
	return slices.Clone(h.mails.sent)
}

type mailRecorder struct {
	mu   sync.Mutex
	sent []Mail
}

var _ mailx.MailerAdapter = (*mailRecorder)(nil)

func (r *mailRecorder) NewMail(context.Context) (mailx.MailInstance, error) {
	return &recordedMail{recorder: r}, nil
}

type recordedMail struct {
	recorder *mailRecorder
	mail     Mail
}

func (m *recordedMail) To(to []string) error {
	m.mail.To = to
	return nil
}

func (m *recordedMail) From(from string) error {
	m.mail.From = from
	return nil
}

func (m *recordedMail) Bcc([]string) error { return nil }

func (m *recordedMail) ReplyTo(string) error { return nil }

func (m *recordedMail) Subject(subject string) error {
	m.mail.Subject = subject
	return nil
}

func (m *recordedMail) HtmlBody(body string) error {
	m.mail.HTMLBody = body
	return nil
}

func (m *recordedMail) TextBody(body string) error {
	m.mail.TextBody = body
	return nil
}

func (m *recordedMail) Send(context.Context) error {
	m.recorder.mu.Lock()
	defer m.recorder.mu.Unlock()
	m.recorder.sent = append(m.recorder.sent, m.mail)
	return nil
}
