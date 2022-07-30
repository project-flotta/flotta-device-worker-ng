package state

import "github.com/tupyy/device-worker-ng/internal/entity"

type profileProcessor struct {
}

func newProfileProcessor(profiles map[string]entity.DeviceProfile) *profileProcessor {
	return &profileProcessor{}
}

func (p *profileProcessor) AddValue(newValue metricValue) {

}

func (p *profileProcessor) Evaluate() map[string]string {
	return map[string]string{}
}
