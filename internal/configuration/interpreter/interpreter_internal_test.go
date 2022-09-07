package interpreter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInterpreter(t *testing.T) {
	exprs := []struct {
		test      string
		expected  bool
		variables map[string]interface{}
		hasError  bool
	}{
		{
			test:     "x == 23",
			expected: true,
			variables: map[string]interface{}{
				"x": 23,
			},
			hasError: false,
		},
		{
			test:     "x == -23.2",
			expected: false,
			variables: map[string]interface{}{
				"x": 23,
			},
			hasError: false,
		},
		{
			test:     "x == 1 && y == 2",
			expected: true,
			variables: map[string]interface{}{
				"x": 1,
				"y": 2,
			},
			hasError: false,
		},
		{
			test:     "(x == 1 && y == 2) || z == 2.2",
			expected: true,
			variables: map[string]interface{}{
				"x": 1,
				"y": 2,
				"z": 1,
			},
			hasError: false,
		},
		{
			test:     "(x == 1 && y == 2) && z == 2.2",
			expected: false,
			variables: map[string]interface{}{
				"x": 1,
				"y": 2,
				"z": 1,
			},
			hasError: false,
		},
		{
			test:     "((x == 1 && y == 2) || z == 2.2) && w == 0",
			expected: true,
			variables: map[string]interface{}{
				"x": 1,
				"y": 2,
				"z": 1,
				"w": 0,
			},
			hasError: false,
		},
		{
			test:     "((x >= 1 && y == 2) || z == 2.2) && w == 0",
			expected: true,
			variables: map[string]interface{}{
				"x": 1,
				"y": 2,
				"z": 1,
				"w": 0,
			},
			hasError: false,
		},
		{
			test:     "x >= 1 && y == 2",
			expected: false,
			variables: map[string]interface{}{
				"x": 0,
				"y": 2,
			},
			hasError: false,
		},
		{
			test:     "x >= 1 && y == 2",
			expected: false,
			variables: map[string]interface{}{
				"x": 0,
			},
			hasError: true,
		},
		{
			test:     "x >= 1 && y == 2",
			expected: true,
			variables: map[string]interface{}{
				"x": 3,
				"y": 2,
			},
			hasError: false,
		},
		{
			test:     "x != 1 && y == 2",
			expected: true,
			variables: map[string]interface{}{
				"x": 3,
				"y": 2,
			},
			hasError: false,
		},
		{
			test:     "x != 1Gib && y == 2%",
			expected: true,
			variables: map[string]interface{}{
				"x": 3,
				"y": 2,
			},
			hasError: false,
		},
		{
			test:     "x == 2 && y == 1 || z == 1",
			expected: true,
			variables: map[string]interface{}{
				"x": 2,
				"y": 1,
				"z": 0,
			},
			hasError: false,
		},
		{
			test:     "x == 2 && y == 1 || z == 1",
			expected: false,
			variables: map[string]interface{}{
				"x": 2,
				"y": 0,
				"z": 0,
			},
			hasError: false,
		},
		{
			test:     "z == 1 || x == 2 && y == 1",
			expected: false,
			variables: map[string]interface{}{
				"x": 2,
				"y": 0,
				"z": 0,
			},
			hasError: false,
		},
		{
			test:     "z == 1 || x == 2 && y == 1 || w == 0",
			expected: true,
			variables: map[string]interface{}{
				"x": 2,
				"y": 0,
				"z": 1,
				"w": 1,
			},
			hasError: false,
		},
		{
			test:      "z == nil",
			expected:  true,
			variables: map[string]interface{}{},
			hasError:  false,
		},
		{
			test:     "z != nil",
			expected: true,
			variables: map[string]interface{}{
				"z": 1,
			},
			hasError: false,
		},
		{
			test:     "z != nil && x == 2",
			expected: true,
			variables: map[string]interface{}{
				"z": 1,
				"x": 2,
			},
			hasError: false,
		},
		{
			test:     "z == nil && x == 2",
			expected: true,
			variables: map[string]interface{}{
				"x": 2,
			},
			hasError: false,
		},
		{
			test:     "z == 2 && x == 2",
			expected: false,
			variables: map[string]interface{}{
				"x": 2,
			},
			hasError: true,
		},
		{
			test:     "z > nil",
			expected: false,
			variables: map[string]interface{}{
				"z": 1,
			},
			hasError: true,
		},
	}

	for idx, data := range exprs {
		t.Run(fmt.Sprintf("test%d: %s", idx+1, data.test), func(t *testing.T) {
			intr, err := New(data.test)
			assert.Nil(t, err)

			if err != nil {
				return
			}

			res, err := intr.Evaluate(data.variables)
			if !data.hasError && err != nil {
				t.Errorf("evaluation error: %v", err)
				return
			}

			assert.Equal(t, data.expected, res)
		})
	}
}
