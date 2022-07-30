package entity

import "github.com/tupyy/device-worker-ng/internal/configuration/interpreter"

/* DeviceProfile specify all the conditions of a profile:
```yaml
state:
	- perfomance:
		- low: cpu<25%
		- medium: cpu>25%
```
In this example the profile is _perfomance_ and the conditions are _low_ and _medium_.
Each condition's expression is evaluated using Variables.
The expression is only evaluated when all the variables need it by the expression are present in the variable map.
*/
type DeviceProfile struct {
	// Name is the name of the profile
	Name string
	// Conditions holds profile's conditions.
	Conditions []ProfileCondition
}

type ProfileCondition struct {
	// Name is the name of the condition
	Name string
	// requiredVariables holds the name of variables required to evaluate the expression
	RequiredVariables []string
	// Expression is the expression's interpreter for the condition
	Expression interpreter.Interpreter
}
