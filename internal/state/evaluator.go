package state

import (
	"github.com/tupyy/device-worker-ng/internal/entity"
)

type profileEvaluator struct {
	Name      string
	Variables map[string]interface{}
	Profile   entity.DeviceProfile
}

// evaluate returns a list of results.
// Each result has map["profile_name.condition_name"] = bool as value or error if there is an evaluation error
func (pe *profileEvaluator) evaluate() ProfileEvaluationResult {
	results := make([]ConditionResult, 0, len(pe.Profile.Conditions))
	for _, condition := range pe.Profile.Conditions {
		res, err := condition.Expression.Evaluate(pe.Variables)
		results = append(results, ConditionResult{
			Name:  condition.Name,
			Value: res,
			Error: err,
		})
	}
	return ProfileEvaluationResult{
		Name:              pe.Name,
		ConditionsResults: results,
	}
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
func (p *simpleEvaluator) Evaluate() entity.Option[[]ProfileEvaluationResult] {
	results := make([]ProfileEvaluationResult, 0, len(p.evaluators))
	for _, e := range p.evaluators {
		results = append(results, e.evaluate())
	}

	return entity.Option[[]ProfileEvaluationResult]{
		Value: results,
		None:  len(results) == 0,
	}
}
