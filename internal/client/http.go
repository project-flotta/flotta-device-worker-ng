package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	rtclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/project-flotta/flotta-operator/client/yggdrasil"
	yggClient "github.com/project-flotta/flotta-operator/client/yggdrasil"
	"github.com/project-flotta/flotta-operator/models"
	config "github.com/tupyy/device-worker-ng/configuration"
	"github.com/tupyy/device-worker-ng/internal/entities"
	"go.uber.org/zap"
)

const (
	certificateKey = "certificate"
)

type HttpClient struct {
	url       *url.URL
	yggClient *yggClient.Client
}

func New(server string, tls *tls.Config) (*HttpClient, error) {
	url, err := url.Parse(server)
	if err != nil {
		return nil, fmt.Errorf("Server address error: %s", err)
	}

	return &HttpClient{yggClient: newYggClient(url, tls), url: url}, nil
}

func newYggClient(url *url.URL, tls *tls.Config) *yggClient.Client {
	host := url.Host
	basePath := url.Path
	schemes := []string{url.Scheme}

	httpTransport := http.Transport{
		ResponseHeaderTimeout: 10 * time.Second,
	}

	if tls != nil {
		httpTransport.TLSClientConfig = tls
		schemes = []string{"https"}
	}

	transport := rtclient.New(host, basePath, schemes)
	transport.Transport = &httpTransport

	return yggdrasil.New(transport, strfmt.Default, nil)
}

func (h *HttpClient) UpdateTLS(newTls *tls.Config) {
	h.yggClient = newYggClient(h.url, newTls)
}

func (h *HttpClient) Enrol(ctx context.Context, enrolInfo entities.EnrolementInfo) error {
	data := yggdrasil.PostDataMessageForDeviceParams{
		DeviceID: config.GetDeviceID(),
		Message: &models.Message{
			Directive: "enrolment",
			Content:   enrolInfoEntity2Model(enrolInfo),
			MessageID: uuid.New().String(),
		},
	}

	zap.S().Debugw("enrol data", "data", data)

	// TODO do something with output
	_, _, err := h.yggClient.PostDataMessageForDevice(ctx, &data)
	if err != nil {
		return err
	}

	return nil
}

func (h *HttpClient) Register(ctx context.Context, registerInfo entities.RegistrationInfo) ([]byte, error) {
	m := registerInfoEntity2Model(registerInfo)

	data := yggdrasil.PostDataMessageForDeviceParams{
		DeviceID: config.GetDeviceID(),
		Message: &models.Message{
			Directive: "registration",
			Content:   m,
			MessageID: uuid.NewString(),
		},
	}

	zap.S().Debugw("registration data", "csr", registerInfo.CertificateRequest, "hardware", registerInfo.Hardware)

	res, _, err := h.yggClient.PostDataMessageForDevice(ctx, &data)
	if err != nil {
		return []byte{}, err
	}

	c, ok := res.Payload.Content.(map[string]interface{})
	if !ok {
		return []byte{}, fmt.Errorf("payload content is not a map")
	}

	cert, ok := c[certificateKey]
	if !ok {
		return []byte{}, fmt.Errorf("cannot get certificate from payload")
	}

	return bytes.NewBufferString(cert.(string)).Bytes(), nil
}

func (h *HttpClient) Heartbeat(ctx context.Context, heartbeat entities.Heartbeat) error {
	zap.S().Debugw("Heartbeat", "heartbeat", heartbeat)
	return nil
}

func (h *HttpClient) GetConfiguration(ctx context.Context) (entities.DeviceConfiguration, error) {
	zap.S().Debugw("Get configuration")
	return entities.DeviceConfiguration{}, nil
}
