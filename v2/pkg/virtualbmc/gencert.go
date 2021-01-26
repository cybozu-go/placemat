package virtualbmc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func generateCertificate(host, outDir string, validFor time.Duration) (string, string, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key: %w", err)
	}

	keyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "placemat",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsAliases(host),
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %w", err)
	}

	_, err = os.Stat(outDir)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		err = os.MkdirAll(outDir, 0755)
		if err != nil {
			return "", "", fmt.Errorf("failed to create output directory: %w", err)
		}
	default:
		return "", "", fmt.Errorf("stat %s failed: %w", outDir, err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	cert := filepath.Join(outDir, "cert.pem")
	if err := outputPEM(cert, "CERTIFICATE", certBytes); err != nil {
		return "", "", err
	}
	key := filepath.Join(outDir, "key.pem")
	if err := outputPEM(key, "PRIVATE KEY", privBytes); err != nil {
		return "", "", err
	}

	return cert, key, nil
}

func dnsAliases(host string) []string {
	parts := strings.Split(host, ".")
	aliases := make([]string, len(parts))
	for i := 0; i < len(parts); i++ {
		aliases[i] = strings.Join(parts[0:len(parts)-i], ".")
	}
	return aliases
}

func outputPEM(fname string, pemType string, data []byte) error {
	f, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", fname, err)
	}
	defer f.Close()

	err = pem.Encode(f, &pem.Block{Type: pemType, Bytes: data})
	if err != nil {
		return fmt.Errorf("failed to encode: %w", err)
	}

	err = f.Sync()
	if err != nil {
		return fmt.Errorf("failed to fsync: %w", err)
	}

	return nil
}
