package state

import (
	"fmt"

	"github.com/tupyy/device-worker-ng/internal/entity"
)

type profileEvaluator struct {
	Name      string
	Variables map[string]interface{}
	Profile   entity.DeviceProfile
}

// evaluate returns a list of results.
// Each result has map["profile_name.condition_name"] = bool as value or error if there is an evaluation error
func (pe *profileEvaluator) evaluate() []EvaluationResult {
	results := make([]EvaluationResult, 0, len(pe.Profile.Conditions))
	for _, condition := range pe.Profile.Conditions {
		res, err := condition.Expression.Evaluate(pe.Variables)
		results = append(results, EvaluationResult{
			Value: map[string]bool{
				fmt.Sprintf("%s.%s", pe.Name, condition.Name): res,
			},
			Error: err,
		})
	}
	return results
}

type simpleEvaluator struct {
	evaluators []*profileEvaluator
}

func (p *simpleEvaluator) SetProfiles(profiles map[string]entity.DeviceProfile) {
	p.evaluators = make([]*profileEvaluator, 0, len(profiles))
	for k, v := range profiles {
		e := profileEvaluator{
			Name:      k,
			Profile:   v,
			Variables: make(map[string]interface{}),
		}
		p.evaluators = append(p.evaluators, &e)
	}
}

func (p *simpleEvaluator) AddValue(newValue metricValue) {
	for _, e := range p.evaluators {
		e.Variables[newValue.Name] = newValue.Value
	}
}

// Evaluate return a list of results for each profile
// each profile can be evaluated to bool or error if the there is a ExpressionError.
func (p *simpleEvaluator) Evaluate() entity.Option[[]EvaluationResult] {
	results := make([]EvaluationResult, 0, len(p.evaluators))
	for _, e := range p.evaluators {
		results = append(results, e.evaluate()...)
	}

	return entity.Option[[]EvaluationResult]{
		Value: results,
		None:  len(results) == 0,
	}
}
