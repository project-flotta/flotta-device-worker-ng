package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/project-flotta/flotta-operator/models"
	"github.com/tupyy/device-worker-ng/internal/entities"
)

type requestType int

const (
	postDataMessageForDeviceType requestType = iota
	getDataMessageForDeviceType
)

type actionType int

const (
	enrolActionType actionType = iota
	registerActionType
	heartbeatActionType
	configurationActionType
)

type RequestBuilder struct {
	request     *http.Request
	action      actionType
	requestType requestType
	body        interface{}
	url         string
	header      map[string]string
}

func newRequestBuilder() *RequestBuilder {
	return &RequestBuilder{
		header: make(map[string]string),
	}
}

func (rb *RequestBuilder) Type(t requestType) *RequestBuilder {
	rb.requestType = t
	return rb
}

func (rb *RequestBuilder) Action(a actionType) *RequestBuilder {
	rb.action = a
	return rb
}

func (rb *RequestBuilder) Url(url string) *RequestBuilder {
	rb.url = url
	return rb
}

func (rb *RequestBuilder) Body(b interface{}) *RequestBuilder {
	rb.body = b
	return rb
}

func (rb *RequestBuilder) Header(key, value string) *RequestBuilder {
	rb.header[key] = value
	return rb
}

func (rb *RequestBuilder) Build(ctx context.Context) (*http.Request, error) {
	var method string
	switch rb.requestType {
	case postDataMessageForDeviceType:
		method = http.MethodPost
	case getDataMessageForDeviceType:
		fallthrough
	default:
		method = http.MethodGet
	}

	var data interface{}

	switch rb.action {
	case enrolActionType:
		enrolInfo, ok := rb.body.(entities.EnrolementInfo)
		if !ok {
			return nil, errors.New("EnrolmentInfo type body is required for this type of request")
		}

		data = &models.Message{
			Directive: "enrolment",
			Content:   enrolInfoEntity2Model(enrolInfo),
			MessageID: uuid.New().String(),
			Type:      "command",
		}
	case registerActionType:
		registrationInfo, ok := rb.body.(entities.RegistrationInfo)
		if !ok {
			return nil, errors.New("RegistrationInfo type body is required for this type of request")
		}

		data = &models.Message{
			Directive: "registration",
			Content:   registerInfoEntity2Model(registrationInfo),
			MessageID: uuid.NewString(),
			Type:      "command",
		}
	case heartbeatActionType:
		heartbeatInfo, ok := rb.body.(entities.Heartbeat)
		if !ok {
			return nil, errors.New("Heartbeat type body is required for this type of request")
		}

		data = &models.Message{
			MessageID: uuid.NewString(),
			Directive: "heartbeat",
			Content:   heartbeatEntity2Model(heartbeatInfo),
		}
	case configurationActionType:
		data = nil
	default:
		return nil, errors.New("unknown request type")
	}

	var body io.ReadCloser
	if data != nil {
		payload, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("cannot marshal '%+v' for enrol action type '%w'", data, err)
		}

		body = io.NopCloser(bytes.NewBuffer(payload))
	}

	request, err := http.NewRequestWithContext(ctx, method, rb.url, body)
	if err != nil {
		return nil, fmt.Errorf("cannot create request '%w'", err)
	}

	for k, v := range rb.header {
		request.Header.Add(k, v)
	}

	return request, nil
}
