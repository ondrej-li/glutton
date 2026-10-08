package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// TokenProvider is anything that can create and validate token based on handler (usually URI) and timestamp.
type TokenProvider interface {
	GenerateToken(handler string, timestamp time.Time) (string, error)
	ValidateToken(token, handler string, timestamp time.Time) (bool, error)
}

// DefaultTokenProvider is the defaul implementation of the TokenProvider interface.
type DefaultTokenProvider struct {
	maximumTokenDuration time.Duration
	key                  []byte
	debug                bool
}

// NewDefaultTokenProvider creates a new instance of the default TokenProvider interface implementation.
func NewDefaultTokenProvider(maximumTokenDuration time.Duration, key []byte, debug bool) *DefaultTokenProvider {
	if len(key) == 0 {
		key = []byte("default-key-change-it-please-now")
	}
	return &DefaultTokenProvider{
		maximumTokenDuration: maximumTokenDuration,
		key:                  key,
		debug:                debug,
	}
}

// GenerateToken token based on provided input string and timestamp.
func (d *DefaultTokenProvider) GenerateToken(handler string, timestamp time.Time) (string, error) {
	plainToken := fmt.Sprintf("%s::%d", handler, timestamp.Unix())
	log.Printf("%s", plainToken)
	encoded, err := encrypt(d.key, plainToken)
	return encoded, err
}

// ValidateToken decodes toekn and returns true if the token is (still) valid.
func (d *DefaultTokenProvider) ValidateToken(token, handler string, timestamp time.Time) (bool, error) {
	plainToken, err := decrypt(d.key, token)
	if err != nil {
		return false, err
	}
	parts := strings.Split(plainToken, "::")
	if len(parts) != 2 {
		return false, errors.New("invalid token")
	}
	tokenHandler := parts[0]
	tokenUnix, _ := strconv.ParseInt(parts[1], 10, 64)
	tokenTimestamp := time.Unix(tokenUnix, 0)
	if tokenHandler != handler {
		log.Printf("token contains handler %s but expected %s", tokenHandler, handler)
		return false, nil
	}
	if !tokenTimestamp.Add(d.maximumTokenDuration).After(timestamp) {
		log.Printf("token timestamp %s expired at %s + %s", tokenTimestamp.String(), timestamp.String(), d.maximumTokenDuration.String())
		return false, nil
	}
	if tokenTimestamp.After(timestamp.Add(time.Second)) {
		log.Printf("token timestamp %s is in the future at %s", tokenTimestamp.String(), timestamp.String())
		return false, nil
	}
	if d.debug {
		log.Printf("token %+v is valid", plainToken)
	}
	return true, nil
}

func encrypt(key []byte, message string) (string, error) {
	plainText := []byte(message)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// add IV
	cipherText := make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)

	return base64.URLEncoding.EncodeToString(cipherText), nil
}

func decrypt(key []byte, securemess string) (string, error) {
	cipherText, err := base64.URLEncoding.DecodeString(securemess)
	if err != nil {
		return "", err

	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err

	}

	if len(cipherText) < aes.BlockSize {
		err = errors.New("ciphertext block size is too short")
		return "", err
	}

	// extract IV
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(cipherText, cipherText)

	return string(cipherText), nil
}
