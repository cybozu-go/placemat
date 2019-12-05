package placemat

import (
	"io/ioutil"
	"os"
)

// CertificateSpec represents a Certificate specification in YAML
type CertificateSpec struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Key  string `json:"key"`
	Cert string `json:"cert"`
}

// Certificate type
type Certificate struct {
	Name         string
	Key          string
	Cert         string
	KeyFilePath  string
	CertFilePath string
}

func (c *Certificate) saveFile() error {
	keyFile, err := ioutil.TempFile("/tmp", "placemat-certificate-key")
	if err != nil {
		return err
	}
	defer keyFile.Close()

	_, err = keyFile.Write([]byte(c.Key))
	if err != nil {
		return err
	}
	c.KeyFilePath = keyFile.Name()

	certFile, err := ioutil.TempFile("/tmp", "placemat-certificate-cert")
	if err != nil {
		return err
	}
	defer certFile.Close()

	_, err = certFile.Write([]byte(c.Cert))
	if err != nil {
		return err
	}
	c.CertFilePath = certFile.Name()
	return nil
}

// CleanupCertificates removes all remaining certificate files.
func CleanupCertificates(r *Runtime, certs []*Certificate) error {
	for _, c := range certs {
		if err := os.Remove(c.CertFilePath); err != nil {
			return err
		}
		if err := os.Remove(c.KeyFilePath); err != nil {
			return err
		}
	}
	return nil
}

// NewCertificate create a new certificate resource
func NewCertificate(spec *CertificateSpec) *Certificate {
	return &Certificate{
		Name: spec.Name,
		Key:  spec.Key,
		Cert: spec.Cert,
	}
}
