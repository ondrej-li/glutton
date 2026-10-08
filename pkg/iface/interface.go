package iface

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ErrPayloadTooLarge is returned by a parser when the request payload exceeds the maximum configured size.
var ErrPayloadTooLarge = errors.New("payload too large")

// PayloadRecord holds data related to a single request in a parsed form.
type PayloadRecord struct {
	// usually the string data of the request
	Payload string
	// timestamp at the time when the payload was created (not request received)
	Timestamp time.Time
	// meta information, things like content-type and so on.
	Meta map[string][]string
	// remote address if provided (may be completely wrong if behind firewalls, proxies etc.)
	Remote string
}

func (p *PayloadRecord) String() string {
	builder := new(strings.Builder)
	builder.WriteString(p.Timestamp.String())
	builder.WriteString(":")
	builder.WriteString(p.Remote)
	builder.WriteString("\n\n")
	builder.WriteString(p.Payload)
	builder.WriteString("\n\n")
	builder.WriteString(fmt.Sprintf("%+v\n", p.Meta))
	return builder.String()
}

// Configurable is anything that can be configured with settings.
type Configurable interface {
	Configure(*Settings) error
}

// PayloadParser is anything that can parse http request and return some payload
type PayloadParser interface {
	Configurable
	Parse(*http.Request) (*PayloadRecord, error)
}

// PayloadSaver can save payload (e.g. to filesystem)
type PayloadSaver interface {
	Configurable
	Save(*PayloadRecord) error
}

// PayloadNotifier is anything that can notify (e.g. send an email, slack) of payload received.
type PayloadNotifier interface {
	Configurable
	Notify(*PayloadRecord) error
}
