package state

import (
	"context"
	"time"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"go.uber.org/zap"
)

type MetricServer interface {
	OutputChannel() chan metricValue
	Shutdown(ctx context.Context) error
}

type Evaluator interface {
	AddValue(newValue metricValue)
	// Evaluate returns a map with profiles which changed state
	// the key is the profile name and value the new state
	Evaluate() (map[string]string, error)
}

type Manager struct {
	// profile condition updates are written to this channel
	OutputCh chan map[string]string

	// profileEvaluator try to determine if a profile changed state
	// after each new metricValue
	profileEvaluator Evaluator

	deviceProfiles map[string]entity.DeviceProfile
	recv           chan entity.Option[map[string]entity.DeviceProfile]
	cancelFunc     context.CancelFunc
	metricServer   MetricServer
}

func New(recv chan entity.Option[map[string]entity.DeviceProfile]) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		OutputCh:   make(chan map[string]string),
		recv:       recv,
		cancelFunc: cancel,
	}

	go m.run(ctx)

	return m
}

func (m *Manager) run(ctx context.Context) {
	var metricChannel chan metricValue

	ticker := time.NewTicker(2 * time.Second)

	for {
		select {
		case opt := <-m.recv:
			// if map empty stop the metric server
			if opt.None {
				if m.metricServer != nil {
					m.metricServer.Shutdown(context.Background())
					metricChannel = nil
					// stop the ticker since we don't have profiles anymore
					ticker.Stop()
					zap.S().Info("metric server stopped")
				}
				break
			}

			zap.S().Info("profile processor created")
			m.profileEvaluator = newProfileEvaluator(opt.Value)

			if m.metricServer == nil {
				m.metricServer = newMetricServer()
				metricChannel = m.metricServer.OutputChannel()
				ticker.Reset(2 * time.Second)
				zap.S().Info("metric server started")
			}
		case metricValue := <-metricChannel:
			zap.S().Debugw("new metric received", "value", metricValue)
			m.profileEvaluator.AddValue(metricValue)
		case <-ticker.C:
			results, err := m.profileEvaluator.Evaluate()
			if err != nil {
				m.OutputCh <- results
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) Shutdown() {
	if m.metricServer != nil {
		m.metricServer.Shutdown(context.Background())
	}
	m.cancelFunc()
}
