package notifier

import (
	"bufio"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/defectus/glutton/pkg/iface"

	"github.com/stretchr/testify/assert"
)

func TestSMTPNotifier_Notify1(t *testing.T) {
	if len(os.Getenv("SMTP_SERVER")) == 0 {
		t.Skip("make sure required env. variables are set before running this script, skipping")
	}
	settings := &iface.Configuration{}
	smtpNotifier := new(SMTPNotifier)
	err := smtpNotifier.Configure(&settings.Settings[0])
	assert.NoError(t, err)
	log.Printf("smtpNotifier:%+v", smtpNotifier)
	err = smtpNotifier.Notify(&iface.PayloadRecord{
		Payload:   "test payload",
		Timestamp: time.Now(),
		Remote:    "0.0.0.0",
	})
	assert.NoError(t, err)
}

type fakeSMTPServer struct {
	listener net.Listener
	startTLS bool
	mu       sync.Mutex
	message  string
}

func newFakeSMTPServer(t *testing.T, startTLS bool) *fakeSMTPServer {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	server := &fakeSMTPServer{listener: listener, startTLS: startTLS}
	go server.serve()
	return server
}

func (f *fakeSMTPServer) port() string {
	_, port, err := net.SplitHostPort(f.listener.Addr().String())
	if err != nil {
		return ""
	}
	return port
}

func (f *fakeSMTPServer) receivedMessage() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.message
}

func (f *fakeSMTPServer) Close() {
	f.listener.Close()
}

func (f *fakeSMTPServer) serve() {
	connection, err := f.listener.Accept()
	if err != nil {
		return
	}
	defer connection.Close()
	connection.SetDeadline(time.Now().Add(5 * time.Second))
	reader := bufio.NewReader(connection)
	write := func(line string) bool {
		_, err := connection.Write([]byte(line + "\r\n"))
		return err == nil
	}
	if !write("220 fake ESMTP ready") {
		return
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		switch command := strings.ToUpper(strings.TrimSpace(line)); {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			if f.startTLS {
				if !write("250-fake") || !write("250 STARTTLS") {
					return
				}
			} else if !write("250 fake") {
				return
			}
		case strings.HasPrefix(command, "QUIT"):
			write("221 bye")
			return
		case strings.HasPrefix(command, "DATA"):
			if !write("354 end with <CRLF>.<CRLF>") {
				return
			}
			var message strings.Builder
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dataLine, "\r\n") == "." {
					break
				}
				message.WriteString(dataLine)
			}
			f.mu.Lock()
			f.message = message.String()
			f.mu.Unlock()
			if !write("250 queued") {
				return
			}
		default:
			if !write("250 ok") {
				return
			}
		}
	}
}

func TestSMTPNotifierNotifyWithoutTLS(t *testing.T) {
	server := newFakeSMTPServer(t, false)
	defer server.Close()

	notifier := &SMTPNotifier{Server: "127.0.0.1", Port: server.port(), From: "from@example.com", To: "to@example.com", Subject: "test"}
	err := notifier.Notify(&iface.PayloadRecord{Payload: "hello notification", Timestamp: time.Now()})
	assert.NoError(t, err)
	assert.Contains(t, server.receivedMessage(), "hello notification")
}

func TestSMTPNotifierNotifyRequiresTLS(t *testing.T) {
	server := newFakeSMTPServer(t, false)
	defer server.Close()

	notifier := &SMTPNotifier{Server: "127.0.0.1", Port: server.port(), UseTLS: true, From: "from@example.com", To: "to@example.com"}
	err := notifier.Notify(&iface.PayloadRecord{Payload: "x", Timestamp: time.Now()})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "STARTTLS")
}
