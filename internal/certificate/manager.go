package certificate

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
)

const (
	ecPrivateKeyBlockType  = "EC PRIVATE KEY"
	rsaPrivateKeyBlockType = "RSA PRIVATE KEY"
)

type Manager struct {
	cert       *x509.Certificate
	privateKey crypto.PrivateKey
	// csrKey is the key used to create the CSR.
	csrKey         crypto.PrivateKey
	privateKeyType string
	rootCA         *x509.CertPool
}

func New(caRootBlock [][]byte, cert, privateKey []byte) (*Manager, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("cannot copy system certificate pool: %w", err)
	}
	for _, data := range caRootBlock {
		pool.AppendCertsFromPEM(data)
	}

	c := &Manager{
		rootCA: pool,
	}

	if err := c.SetCertificate(cert, privateKey); err != nil {
		return nil, err
	}

	return c, nil
}

// Certificates set a new certificate and a private key.
func (c *Manager) SetCertificate(cert, privateKey []byte) error {
	certPem, _ := pem.Decode(cert)
	if certPem == nil {
		return fmt.Errorf("cannot decode certificate from pem")
	}

	newCert, err := x509.ParseCertificate(certPem.Bytes)
	if err != nil {
		return fmt.Errorf("cannot parse certificate: %w", err)
	}

	// decode key
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return fmt.Errorf("cannot private key")
	}

	var key crypto.Signer

	switch block.Type {
	case ecPrivateKeyBlockType:
		key, err = x509.ParseECPrivateKey(block.Bytes)
	case rsaPrivateKeyBlockType:
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	default:
		err = fmt.Errorf("unknown block type")
	}

	if err != nil {
		return fmt.Errorf("cannot decode private key: %w", err)
	}

	c.cert = newCert
	c.privateKey = key
	c.privateKeyType = block.Type

	return nil
}

func (c *Manager) TLSConfig() (tls.Config, error) {
	config := tls.Config{
		RootCAs: c.rootCA,
	}

	certPEM := new(bytes.Buffer)
	err := pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: c.cert.Raw,
	})
	//
	cert, err := tls.X509KeyPair(certPEM.Bytes(), c.marshalKeyToPem(c.privateKey).Bytes())
	if err != nil {
		return tls.Config{}, fmt.Errorf("cannot create x509 key pair: %w", err)
	}

	config.Certificates = []tls.Certificate{cert}

	return config, nil
}

func (c *Manager) GenerateCSR(deviceID string) ([]byte, []byte, error) {
	var csrTemplate = x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: deviceID,
			// Operator will add metadata on this subject, like namespace
		},
	}

	key, err := c.generateKey(c.privateKeyType)
	if err != nil {
		return nil, nil, err
	}

	csrCertificate, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, key)
	if err != nil {
		return nil, nil, err
	}

	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrCertificate,
	})

	return csrPEM, c.marshalKeyToPem(key).Bytes(), nil
}

func (c *Manager) CommonName() string {
	return c.cert.Subject.CommonName
}

func (c *Manager) WriteCertificate(certPath, keyPath string) error {
	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: c.cert.Raw})
	certOut.Close()

	err = ioutil.WriteFile(keyPath, c.marshalKeyToPem(c.privateKey).Bytes(), 0600)
	if err != nil {
		return err
	}

	return nil
}

func (c *Manager) marshalKeyToPem(key crypto.PrivateKey) *bytes.Buffer {
	privKeyPEM := new(bytes.Buffer)
	switch t := key.(type) {
	case *ecdsa.PrivateKey:
		res, _ := x509.MarshalECPrivateKey(t)
		_ = pem.Encode(privKeyPEM, &pem.Block{
			Type:  ecPrivateKeyBlockType,
			Bytes: res,
		})
	case *rsa.PrivateKey:
		_ = pem.Encode(privKeyPEM, &pem.Block{
			Type:  rsaPrivateKeyBlockType,
			Bytes: x509.MarshalPKCS1PrivateKey(t),
		})
	}

	return privKeyPEM
}

func (c *Manager) generateKey(keyType string) (crypto.PrivateKey, error) {
	switch keyType {
	case ecPrivateKeyBlockType:
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case rsaPrivateKeyBlockType:
		return rsa.GenerateKey(rand.Reader, 4096)
	default:
		return nil, fmt.Errorf("unknown algorithm to create the key")
	}
}
