package common

import (
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/defectus/glutton/pkg/iface"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestServerTLSConfigDisabled(t *testing.T) {
	configuration, err := serverTLSConfig(&iface.Configuration{})
	assert.NoError(t, err)
	assert.Nil(t, configuration)
}

func TestServerTLSConfigSelfSigned(t *testing.T) {
	configuration, err := serverTLSConfig(&iface.Configuration{UseTLS: true, Host: "glutton.example.com"})
	assert.NoError(t, err)
	assert.NotNil(t, configuration)
	assert.Len(t, configuration.Certificates, 1)
	assert.Equal(t, uint16(tls.VersionTLS12), configuration.MinVersion)

	leaf := configuration.Certificates[0].Leaf
	assert.NotNil(t, leaf)
	assert.Contains(t, leaf.DNSNames, "localhost")
	assert.Contains(t, leaf.DNSNames, "glutton.example.com")
	assert.NoError(t, leaf.VerifyHostname("localhost"))
	assert.NoError(t, leaf.VerifyHostname("127.0.0.1"))
	assert.NoError(t, leaf.VerifyHostname("glutton.example.com"))
}

func TestServerTLSConfigSelfSignedWildcardHost(t *testing.T) {
	configuration, err := serverTLSConfig(&iface.Configuration{UseTLS: true, Host: "0.0.0.0"})
	assert.NoError(t, err)
	assert.Len(t, configuration.Certificates, 1)
	assert.Equal(t, []string{"localhost"}, configuration.Certificates[0].Leaf.DNSNames)
}

func TestServerTLSConfigRequiresBothCertificateFiles(t *testing.T) {
	_, err := serverTLSConfig(&iface.Configuration{UseTLS: true, CertFile: "cert.pem"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "both cert_file and key_file")
}

func TestServerTLSConfigFailsOnMissingCertificate(t *testing.T) {
	_, err := serverTLSConfig(&iface.Configuration{UseTLS: true, CertFile: "missing-cert.pem", KeyFile: "missing-key.pem"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error loading certificate")
}

func TestServerTLSConfigWithProvidedCertificate(t *testing.T) {
	certificate, err := selfSignedCertificate("localhost")
	assert.NoError(t, err)
	certFile, keyFile := writeCertificateFiles(t, certificate)

	configuration, err := serverTLSConfig(&iface.Configuration{UseTLS: true, CertFile: certFile, KeyFile: keyFile})
	assert.NoError(t, err)
	assert.NotNil(t, configuration)
	assert.Len(t, configuration.Certificates, 1)
}

func TestServeHTTPS(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	address := listener.Addr().String()
	assert.NoError(t, listener.Close())

	tlsConfiguration, err := serverTLSConfig(&iface.Configuration{UseTLS: true, Host: "127.0.0.1"})
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, router, address, tlsConfiguration)
	}()

	// the generated certificate is self signed, hence the skip verify
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	var lastError error
	for i := 0; i < 30; i++ {
		response, err := client.Get("https://" + address + "/ping")
		if err == nil {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			assert.Equal(t, http.StatusOK, response.StatusCode)
			assert.Equal(t, "pong", string(body))
			lastError = nil
			break
		}
		lastError = err
		time.Sleep(100 * time.Millisecond)
	}
	assert.NoError(t, lastError)

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after the context was cancelled")
	}
}

func writeCertificateFiles(t *testing.T, certificate tls.Certificate) (string, string) {
	directory := t.TempDir()
	certFile := filepath.Join(directory, "cert.pem")
	keyFile := filepath.Join(directory, "key.pem")

	certificateOut, err := os.Create(certFile)
	assert.NoError(t, err)
	defer certificateOut.Close()
	assert.NoError(t, pem.Encode(certificateOut, &pem.Block{Type: "CERTIFICATE", Bytes: certificate.Certificate[0]}))

	key, ok := certificate.PrivateKey.(*ecdsa.PrivateKey)
	assert.True(t, ok)
	keyBytes, err := x509.MarshalECPrivateKey(key)
	assert.NoError(t, err)
	keyOut, err := os.Create(keyFile)
	assert.NoError(t, err)
	defer keyOut.Close()
	assert.NoError(t, pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}))

	return certFile, keyFile
}
