package parser

import (
	"io"
	"net/http"
	"time"

	"github.com/defectus/glutton/pkg/iface"
	"github.com/pkg/errors"
)

// SimpleParser is the default implementation if the parser interface.
type SimpleParser struct {
	maxBodySize int
}

// Parse reads request and builds a payload from it. When a maximum body size is configured the read is bounded and iface.ErrPayloadTooLarge is returned when the body exceeds it.
func (s *SimpleParser) Parse(req *http.Request) (*iface.PayloadRecord, error) {
	payload := &iface.PayloadRecord{}
	reader := io.Reader(req.Body)
	if s.maxBodySize > 0 {
		reader = io.LimitReader(req.Body, int64(s.maxBodySize)+1)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrap(err, "error reading payload")
	}
	if s.maxBodySize > 0 && len(body) > s.maxBodySize {
		return nil, iface.ErrPayloadTooLarge
	}
	payload.Payload = string(body)
	payload.Timestamp = time.Now()
	payload.Remote = req.RemoteAddr
	payload.Meta = req.Header
	return payload, nil
}

// Configure initilizes the instance of parser.
func (s *SimpleParser) Configure(settings *iface.Settings) error {
	s.maxBodySize = settings.MaxBodySize
	return nil
}
