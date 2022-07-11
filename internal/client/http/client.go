package client

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tupyy/device-worker-ng/internal/certificate"
	"github.com/tupyy/device-worker-ng/internal/entities"
	"go.uber.org/zap"
)

const (
	certificateKey = "certificate"
	rootUrl        = "/api/flotta-management/v1"
)

// transportWrapper is a wrapper for transport. It can be used as a middleware.
type transportWrapper func(http.RoundTripper) http.RoundTripper

type Client struct {
	// certMananger holds the Certificate Manager
	certMananger *certificate.Manager

	// certificateSignature holds the signature of the client certificate which is used in TLS config.
	// It is used to check if certificates had been updated following registration process.
	certificateSignature []byte

	// server's url
	serverURL *url.URL

	transportWrappers []transportWrapper

	// transport is the transport which make the actual request
	transport http.RoundTripper
}

func New(path string, certManager *certificate.Manager) (*Client, error) {
	if certManager == nil {
		return nil, fmt.Errorf("Certificate manager is missing")
	}

	url, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("Server address error: %s", err)
	}

	// TODO dynamically set based on log level
	transportWrapper := make([]transportWrapper, 0, 1)
	logWrapper := &logTransportWrapper{}
	transportWrapper = append(transportWrapper, logWrapper.Wrap)

	return &Client{
		serverURL:            url,
		certMananger:         certManager,
		certificateSignature: []byte{},
		transportWrappers:    transportWrapper,
	}, nil
}

func (c *Client) Enrol(ctx context.Context, deviceID string, enrolInfo entities.EnrolementInfo) error {
	request, err := newRequestBuilder().
		Type(postDataMessageForDeviceType).
		Action(enrolActionType).
		Header("Content-Type", "application/json").
		Url(fmt.Sprintf("%s/%s/data/%s/out", c.serverURL.String(), rootUrl, deviceID)).
		Body(enrolInfo).
		Build(ctx)

	if err != nil {
		return fmt.Errorf("cannot create enrollment request '%w'", err)
	}

	response, err := c.do(request)
	if err != nil {
		return fmt.Errorf("cannot enrol device '%w'", err)
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("cannot enrol device. code: %d", response.StatusCode)
	}

	return nil
}

func (c *Client) Register(ctx context.Context, deviceID string, registerInfo entities.RegistrationInfo) (entities.RegistrationResponse, error) {
	request, err := newRequestBuilder().
		Type(postDataMessageForDeviceType).
		Action(registerActionType).
		Header("Content-Type", "application/json").
		Url(fmt.Sprintf("%s/%s/data/%s/out", c.serverURL.String(), rootUrl, deviceID)).
		Body(registerInfo).
		Build(ctx)

	if err != nil {
		return entities.RegistrationResponse{}, fmt.Errorf("cannot create registration request '%w'", err)
	}

	response, err := c.do(request)
	if err != nil {
		return entities.RegistrationResponse{}, fmt.Errorf("cannot register device '%w'", err)
	}

	data, err := extractData(response, certificateKey, func(data string) (string, error) { return data, nil })
	if err != nil {
		return entities.RegistrationResponse{}, err
	}

	return entities.RegistrationResponse{SignedCSR: bytes.NewBufferString(data).Bytes()}, nil
}

func (c *Client) Heartbeat(ctx context.Context, deviceID string, heartbeat entities.Heartbeat) error {
	request, err := newRequestBuilder().
		Type(postDataMessageForDeviceType).
		Action(heartbeatActionType).
		Url(fmt.Sprintf("%s/%s/data/%s/out", c.serverURL.String(), rootUrl, deviceID)).
		Body(heartbeat).
		Header("Content-Type", "application/json").
		Build(ctx)

	if err != nil {
		return fmt.Errorf("cannot create heartbeat request '%w'", err)
	}

	response, err := c.do(request)
	if err != nil {
		return fmt.Errorf("cannot send heartbeat '%w'", err)
	}

	// TODO send typed error based on status code
	if response.StatusCode >= 400 {
		return fmt.Errorf("cannot send heartbeat. code: %d", response.StatusCode)
	}

	return nil
}

func (c *Client) GetConfiguration(ctx context.Context, deviceID string) (entities.DeviceConfiguration, error) {
	request, err := newRequestBuilder().
		Type(getDataMessageForDeviceType).
		Action(configurationActionType).
		Header("Content-Type", "application/json").
		Url(fmt.Sprintf("%s/%s/data/%s/in", c.serverURL.String(), rootUrl, deviceID)).
		Build(ctx)

	if err != nil {
		return entities.DeviceConfiguration{}, fmt.Errorf("cannot create configuration request '%w'", err)
	}

	response, err := c.do(request)
	if err != nil {
		return entities.DeviceConfiguration{}, fmt.Errorf("cannot get configuration '%w'", err)
	}

	// TODO check the response code

	data, err := extractData(response, "configuration", transformToConfiguration)
	if err != nil {
		return entities.DeviceConfiguration{}, err
	}

	return configurationModel2Entity(data), nil
}

func (c *Client) do(request *http.Request) (*http.Response, error) {
	client, err := c.getClient()
	if err != nil {
		return nil, err
	}

	return client.Do(request)
}

// getClient returns a real http.Client created with our transport.
// It checks if certifcates signatures changed and if true it recreates a new transport.
func (c *Client) getClient() (*http.Client, error) {
	if !bytes.Equal(c.certificateSignature, c.certMananger.Signature()) {
		zap.S().Info("Certificates have changed. Recreate transport")
		t, err := c.createTransport()
		if err != nil {
			return nil, err
		}

		c.certificateSignature = c.certMananger.Signature()

		c.transport = t
	}

	return &http.Client{
		Transport: c.transport,
		Timeout:   2 * time.Second, //TODO to be parametrized
	}, nil

}

func (c *Client) createTransport() (result http.RoundTripper, err error) {
	var tlsConfig *tls.Config

	tlsConfig, err = c.createTLSConfig()

	result = &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
	}

	// call the other wrappers backwards
	for i := len(c.transportWrappers) - 1; i >= 0; i-- {
		result = c.transportWrappers[i](result)
	}

	return result, err
}

func (c *Client) createTLSConfig() (*tls.Config, error) {
	caRoot, cert, key := c.certMananger.GetCertificates()

	config := tls.Config{
		RootCAs: caRoot,
	}

	certPEM := new(bytes.Buffer)
	err := pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	privKeyPEM := new(bytes.Buffer)
	switch t := key.(type) {
	case *ecdsa.PrivateKey:
		res, _ := x509.MarshalECPrivateKey(t)
		_ = pem.Encode(privKeyPEM, &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: res,
		})
	case *rsa.PrivateKey:
		_ = pem.Encode(privKeyPEM, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(t),
		})
	}

	//
	cc, err := tls.X509KeyPair(certPEM.Bytes(), privKeyPEM.Bytes())
	if err != nil {
		return nil, fmt.Errorf("cannot create x509 key pair: %w", err)
	}

	config.Certificates = []tls.Certificate{cc}

	return &config, nil
}
