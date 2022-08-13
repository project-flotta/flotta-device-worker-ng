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

type EvaluationResult entity.Result[map[string]bool]

type Evaluator interface {
	SetProfiles(profiles map[string]entity.DeviceProfile)
	AddValue(newValue metricValue)
	// Evaluate returns list of results for each profile.
	// The result is a map having as key the name of the profile and the result as value.
	// If the profile expression evaluates with error, the error in Result is set accordantly.
	Evaluate() entity.Option[[]EvaluationResult]
}

type Manager struct {
	// profile condition updates are written to this channel
	OutputCh chan entity.Message

	// profileEvaluator try to determine if a profile changed state
	// after each new metricValue
	profilesEvaluator Evaluator

	deviceProfiles map[string]entity.DeviceProfile
	recv           chan entity.Message
	cancelFunc     context.CancelFunc
	metricServer   MetricServer
}

// New returns a new state manager with the default evaluator
func New(recv chan entity.Message) *Manager {
	return _new(recv, &simpleEvaluator{})
}

// NewWithEvaluator returns a new state manager with the provided evaluator
func NewWithEvaluator(recv chan entity.Message, e Evaluator) *Manager {
	return _new(recv, e)
}

func _new(recv chan entity.Message, evaluator Evaluator) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		OutputCh:          make(chan entity.Message),
		recv:              recv,
		cancelFunc:        cancel,
		profilesEvaluator: evaluator,
	}

	go m.run(ctx)

	return m
}

func (m *Manager) run(ctx context.Context) {
	var metricChannel chan metricValue

	ticker := time.NewTicker(2 * time.Second)
	input := make(chan entity.Option[map[string]entity.DeviceProfile])

	for {
		select {
		case m := <-m.recv:
			switch m.Kind {
			case entity.ProfileConfigurationMessage:
				val, ok := m.Payload.(entity.Option[map[string]entity.DeviceProfile])
				if !ok {
					zap.S().Errorf("mismatch message payload type. expected workload. got %v", m)
				}
				input <- val
			}
		case opt := <-input:
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
			m.profilesEvaluator.SetProfiles(opt.Value)

			if m.metricServer == nil {
				m.metricServer = newMetricServer()
				metricChannel = m.metricServer.OutputChannel()
				ticker.Reset(2 * time.Second)
				zap.S().Info("metric server started")
			}
		case metricValue := <-metricChannel:
			zap.S().Debugw("new metric received", "value", metricValue)
			m.profilesEvaluator.AddValue(metricValue)
		case <-ticker.C:
			opt := m.profilesEvaluator.Evaluate()
			if opt.None {
				break
			}

			zap.S().Debugw("evaluate profiles", "results", opt.Value)
			m.OutputCh <- entity.Message{
				Kind:    entity.ProfileConfigurationMessage,
				Payload: opt.Value,
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
