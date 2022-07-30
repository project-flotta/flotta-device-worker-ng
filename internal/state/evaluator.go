package state

import "github.com/tupyy/device-worker-ng/internal/entity"

type profileEvaluator struct {
}

func newProfileEvaluator(profiles map[string]entity.DeviceProfile) *profileEvaluator {
	return &profileEvaluator{}
}

func (p *profileEvaluator) AddValue(newValue metricValue) {

}

func (p *profileEvaluator) Evaluate() (map[string]string, error) {
	return map[string]string{}, nil
}
