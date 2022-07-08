package client

import (
	"bytes"
	"context"
	"io/ioutil"
	"mime"
	"net/http"

	"go.uber.org/zap"
)

type logTransportWrapper struct {
	next http.RoundTripper
}

func (l *logTransportWrapper) Wrap(transport http.RoundTripper) http.RoundTripper {
	return &logTransportWrapper{
		next: transport,
	}
}

func (l *logTransportWrapper) RoundTrip(request *http.Request) (response *http.Response, err error) {
	ctx := request.Context()

	// Read the complete body in memory, in order to send it to the log, and replace it with a
	// reader that reads it from memory:
	if request.Body != nil {
		var body []byte
		body, err = ioutil.ReadAll(request.Body)
		if err != nil {
			return
		}

		err = request.Body.Close()
		if err != nil {
			return
		}

		l.logRequest(ctx, request, body)
		request.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	} else {
		l.logRequest(ctx, request, nil)
	}

	// Call the next round tripper
	response, err = l.next.RoundTrip(request)
	if err != nil {
		return
	}

	// Read the complete response body in memory, in order to send it the log, and replace it
	// with a reader that reads it from memory:
	if response.Body != nil {
		var body []byte
		body, err = ioutil.ReadAll(response.Body)
		if err != nil {
			return
		}

		err = response.Body.Close()
		if err != nil {
			return
		}

		l.logResponse(ctx, response, body)
		response.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	} else {
		l.logResponse(ctx, response, nil)
	}

	return
}

func (l *logTransportWrapper) logRequest(ctx context.Context, request *http.Request, body []byte) {
	zap.S().Debugf("Request method is: %s", request.Method)
	zap.S().Debugf("Request URL is: %s", request.URL)

	if request.Host != "" {
		zap.S().Debugf("Request host: %s", request.Host)
	}

	header := request.Header
	for k, values := range header {
		for _, value := range values {
			zap.S().Debugw("Response header", "key", k, "value", value)
		}
	}

	if body != nil {
		zap.S().Debug("Request body follows")
		l.logBody(ctx, header, body)
	}
}

func (l *logTransportWrapper) logResponse(ctx context.Context, response *http.Response, body []byte) {
	zap.S().Debugf("Response protocol is: %s", response.Proto)
	zap.S().Debugf("Response status is: %s", response.Status)

	header := response.Header
	for k, values := range header {
		for _, value := range values {
			zap.S().Debugw("Response header", "key", k, "value", value)
		}
	}

	if body != nil {
		zap.S().Debug("Response body follows")
		l.logBody(ctx, header, body)
	}
}

func (l *logTransportWrapper) logBody(ctx context.Context, header http.Header, body []byte) {
	// Try to parse the content type:
	var mediaType string
	contentType := header.Get("Content-Type")
	if contentType != "" {
		var err error
		mediaType, _, err = mime.ParseMediaType(contentType)
		if err != nil {
			zap.S().Errorf("Can't parse content type '%s': %v", contentType, err)
		}
	} else {
		mediaType = contentType
	}

	// Dump the body according to the content type:
	switch mediaType {
	case "application/json", "":
		fallthrough
	default:
		zap.S().Debugf("%s", body)
	}
}
