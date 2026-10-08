package handler

import (
	"crypto/subtle"
	"log"
	"net/http"
	"time"

	"github.com/defectus/glutton/pkg/auth"
	"github.com/defectus/glutton/pkg/iface"
	"github.com/gin-gonic/gin"
)

// TokenKeyHeader is the header a caller must set to the configured token key in order to obtain an access token.
const TokenKeyHeader = "token-key"

// CreateTokenHandler setups a handler that returns access tokens. The caller has to present the configured token key in the TokenKeyHeader header.
func CreateTokenHandler(uri string, key []byte, debug bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if subtle.ConstantTimeCompare([]byte(c.GetHeader(TokenKeyHeader)), key) != 1 {
			log.Printf("token requested for %s without a valid %s header", uri, TokenKeyHeader)
			c.Status(http.StatusPreconditionFailed)
			return
		}
		tokenProvider := auth.NewDefaultTokenProvider(5*time.Minute, key, debug)
		token, err := tokenProvider.GenerateToken(uri, time.Now())
		if err != nil {
			log.Printf("error generating token %+v", err)
		}
		c.Writer.Write([]byte(token))
	}
}

// RedirectHandler wraps the supplied handler into a redirect call. If the location parameters is empty no redirect is performed.
func RedirectHandler(h gin.HandlerFunc, code int, location string) gin.HandlerFunc {
	if len(location) == 0 {
		return h
	}
	return func(c *gin.Context) {
		h(c)
		// do not redirect when the wrapped handler already reported a failure
		if c.Writer.Status() >= http.StatusBadRequest {
			return
		}
		c.Redirect(code, location)
	}
}

// ValidateTokenHandler wraps target handler into token validation block.
func ValidateTokenHandler(h gin.HandlerFunc, uri string, key []byte, debug bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenProvider := auth.NewDefaultTokenProvider(5*time.Minute, key, debug)
		valid, err := tokenProvider.ValidateToken(c.GetHeader("token"), uri, time.Now())
		if err != nil {
			log.Printf("a valid token required but validation failed with %+v", err)
		}
		if !valid {
			c.Status(http.StatusPreconditionFailed)
			return
		}
		h(c)
	}
}

// CreateHandler appends a route to router and initialize the basic flow (request -> parser -> notifier -> saver)
func CreateHandler(URI string, parser iface.PayloadParser, notifier iface.PayloadNotifier, saver iface.PayloadSaver, debug bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, err := parser.Parse(c.Request)
		if err != nil {
			log.Printf("%s: error parsing contents %+v", URI, err)
			log.Printf("%+v", c.Request)
		}
		err = notifier.Notify(payload)
		if err != nil {
			log.Printf("%s: error notifying of payload %+v", URI, err)
			log.Printf("%+v", payload)
		}
		err = saver.Save(payload)
		if err != nil {
			log.Printf("%s: error saving payload %+v", URI, err)
			log.Printf("%+v", payload)
		}
		c.Status(http.StatusOK)
	}
}
