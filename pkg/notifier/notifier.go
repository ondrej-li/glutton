package notifier

import (
	"bytes"
	"crypto/tls"
	"html/template"
	"log"
	"net"
	"net/smtp"
	"time"

	"github.com/defectus/glutton/pkg/iface"
	"github.com/pkg/errors"
)

// smtpDialTimeout is the maximum time allowed to establish the connection to the SMTP server.
const smtpDialTimeout = 10 * time.Second

// NilNotifier implements the glutton.PayloadNotifier interface but does nothing.
type NilNotifier struct{}

// SMTPNotifier implements the glutton.PayloadNotifier interface and delivers notifications over SMTP.
type SMTPNotifier struct {
	Server   string
	Port     string
	UseTLS   bool
	From     string
	Password string
	To       string
	Subject  string
}

// Notify does nothing.
func (n *NilNotifier) Notify(*iface.PayloadRecord) error {
	return nil
}

// Configure does nothing.
func (n *NilNotifier) Configure(*iface.Settings) error {
	return nil
}

// PayloadToSMTPMessage takes payload and format it to SMTP format.
func (s *SMTPNotifier) PayloadToSMTPMessage(payload *iface.PayloadRecord) []byte {
	smtpTemplateData := &struct {
		From    string
		To      string
		Subject string
		Body    string
	}{s.From, s.To, s.Subject, payload.String()}
	const emailTemplate = `From: {{.From}}
To: {{.To}}
Subject: {{.Subject}}

{{.Body}}

Sincerely,

{{.From}}
`
	var err error
	var doc bytes.Buffer

	t := template.New("emailTemplate")
	t, err = t.Parse(emailTemplate)
	if err != nil {
		log.Printf("error trying to parse mail template %+v", err)
	}
	err = t.Execute(&doc, smtpTemplateData)
	if err != nil {
		log.Printf("error trying to execute mail template %+v", err)
	}
	return doc.Bytes()
}

// Notify sends notification over SMTP. When UseTLS is set the server has to support STARTTLS, otherwise the notification is not sent - this avoids a silent downgrade to an unencrypted connection.
func (s *SMTPNotifier) Notify(payload *iface.PayloadRecord) error {
	address := s.Server + ":" + s.Port
	connection, err := net.DialTimeout("tcp", address, smtpDialTimeout)
	if err != nil {
		return errors.Wrapf(err, "error connecting to %s", address)
	}
	defer connection.Close()

	client, err := smtp.NewClient(connection, s.Server)
	if err != nil {
		return errors.Wrap(err, "error creating smtp client")
	}
	defer client.Close()

	if s.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.Errorf("server %s does not support STARTTLS but tls is required", address)
		}
		if err := client.StartTLS(&tls.Config{ServerName: s.Server}); err != nil {
			return errors.Wrap(err, "error starting tls")
		}
	}
	if ok, _ := client.Extension("AUTH"); ok {
		if err := client.Auth(smtp.PlainAuth("", s.From, s.Password, s.Server)); err != nil {
			return errors.Wrap(err, "error authenticating")
		}
	}
	if err := client.Mail(s.From); err != nil {
		return errors.Wrap(err, "error setting sender")
	}
	if err := client.Rcpt(s.To); err != nil {
		return errors.Wrap(err, "error setting recipient")
	}
	writer, err := client.Data()
	if err != nil {
		return errors.Wrap(err, "error starting message data")
	}
	if _, err := writer.Write(s.PayloadToSMTPMessage(payload)); err != nil {
		return errors.Wrap(err, "error writing message")
	}
	if err := writer.Close(); err != nil {
		return errors.Wrap(err, "error closing message")
	}
	return errors.Wrap(client.Quit(), "error sending notification")
}

// Configure configures this notifier according to settings.
func (s *SMTPNotifier) Configure(settings *iface.Settings) error {
	s.Server = settings.SMTPServer
	s.Port = settings.SMTPPort
	s.UseTLS = settings.SMTPUseTLS
	s.From = settings.SMTPFrom
	s.Password = settings.SMTPPassword
	s.To = settings.SMTPTo
	s.Subject = "Notification from " + settings.Name
	return nil
}
