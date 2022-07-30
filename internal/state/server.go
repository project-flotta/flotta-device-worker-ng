package state

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync/atomic"
)

type metricValue struct {
	Value float32 `json:"value"`
	Name  string  `json:"name"`
}

// this is a http implementation but it could be anything.
type metricServer struct {
	outCh  chan metricValue
	count  int32
	max    int32
	server http.Server
	doneCh chan struct{}
}

func newMetricServer() MetricServer {
	m := &metricServer{
		outCh: make(chan metricValue, 1),
		count: 0,
		max:   10,
		server: http.Server{
			Addr: ":8080",
		},
		doneCh: make(chan struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", m.handler())
	m.server.Handler = mux

	go func() {
		<-m.doneCh
		m.server.Shutdown(context.Background())
	}()

	go func() {
		m.server.ListenAndServe()
	}()

	return m
}

func (m *metricServer) OutputChannel() chan metricValue {
	return m.outCh
}

func (m *metricServer) Shutdown(ctx context.Context) error {
	m.doneCh <- struct{}{}
	return nil
}

func (m *metricServer) handler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&m.count) >= m.max {
			w.WriteHeader(http.StatusOK)
			return
		}

		d, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		defer r.Body.Close()

		var data metricValue
		if err := json.Unmarshal(d, &data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		atomic.AddInt32(&m.count, 1)
		go func() {
			m.outCh <- data
			atomic.AddInt32(&m.count, -1)
		}()

		w.WriteHeader(http.StatusOK)
	}
}
