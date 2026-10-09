package common

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"log"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/defectus/glutton/pkg/iface"
	"github.com/pkg/errors"
)

// selfSignedValidity is how long a generated certificate is valid for.
const selfSignedValidity = 365 * 24 * time.Hour

// serverTLSConfig builds the TLS configuration of the server. It returns nil when TLS is not enabled, the certificate provided through settings when both files are given, and a freshly generated self signed certificate otherwise.
func serverTLSConfig(configuration *iface.Configuration) (*tls.Config, error) {
	if !configuration.UseTLS {
		return nil, nil
	}
	var (
		certificate tls.Certificate
		err         error
	)
	switch {
	case len(configuration.CertFile) > 0 && len(configuration.KeyFile) > 0:
		certificate, err = tls.LoadX509KeyPair(configuration.CertFile, configuration.KeyFile)
		if err != nil {
			return nil, errors.Wrapf(err, "error loading certificate %s and key %s", configuration.CertFile, configuration.KeyFile)
		}
	case len(configuration.CertFile) > 0 || len(configuration.KeyFile) > 0:
		return nil, errors.New("both cert_file and key_file have to be configured to use a provided certificate")
	default:
		certificate, err = selfSignedCertificate(configuration.Host)
		if err != nil {
			return nil, errors.Wrap(err, "error generating self signed certificate")
		}
		log.Printf("tls is enabled without a certificate, using a generated self signed one valid for %s", strings.Join(certificateHosts(configuration.Host), ", "))
	}
	return &tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// selfSignedCertificate generates an in memory self signed certificate for the given host.
func selfSignedCertificate(host string) (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "error generating key")
	}
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "error generating serial number")
	}
	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: "glutton"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(selfSignedValidity),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	template.DNSNames, template.IPAddresses = certificateNames(host)
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "error creating certificate")
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: &template}, nil
}

// certificateNames returns the DNS names and IP addresses a generated certificate is valid for.
func certificateNames(host string) ([]string, []net.IP) {
	names := []string{"localhost"}
	addresses := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
	switch host {
	case "", "0.0.0.0", "::":
		// listening on every interface, the loopback names above cover it
	case "localhost", "127.0.0.1", "::1":
		// already covered
	default:
		if address := net.ParseIP(host); address != nil {
			addresses = append(addresses, address)
		} else {
			names = append(names, host)
		}
	}
	return names, addresses
}

func certificateHosts(host string) []string {
	names, addresses := certificateNames(host)
	hosts := append([]string{}, names...)
	for _, address := range addresses {
		hosts = append(hosts, address.String())
	}
	return hosts
}
