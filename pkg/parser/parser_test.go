package parser

import (
	"net/http"
	"strings"
	"testing"

	"github.com/defectus/glutton/pkg/iface"
	"github.com/stretchr/testify/assert"
)

func parseRequest(t *testing.T, maxBodySize int, body string) (*iface.PayloadRecord, error) {
	p := new(SimpleParser)
	assert.NoError(t, p.Configure(&iface.Settings{MaxBodySize: maxBodySize}))
	req, err := http.NewRequest("POST", "http://localhost/save", strings.NewReader(body))
	assert.NoError(t, err)
	return p.Parse(req)
}

func TestSimpleParserParse(t *testing.T) {
	payload, err := parseRequest(t, 1024, "hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello", payload.Payload)
	assert.NotZero(t, payload.Timestamp)
}

func TestSimpleParserAcceptsPayloadAtLimit(t *testing.T) {
	payload, err := parseRequest(t, 5, "hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello", payload.Payload)
}

func TestSimpleParserRejectsOversizedPayload(t *testing.T) {
	_, err := parseRequest(t, 4, "hello world")
	assert.Equal(t, iface.ErrPayloadTooLarge, err)
}

func TestSimpleParserWithoutLimit(t *testing.T) {
	payload, err := parseRequest(t, 0, strings.Repeat("x", 10000))
	assert.NoError(t, err)
	assert.Len(t, payload.Payload, 10000)
}
