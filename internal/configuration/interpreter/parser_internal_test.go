package interpreter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	exprs := []struct {
		test     string
		expected string
		hasError bool
	}{
		{
			test:     "x == 23",
			expected: "( x == 23 )",
			hasError: false,
		},
		{
			test:     "x == -23.2",
			expected: "( x == -23.2 )",
			hasError: false,
		},
		{
			test:     "x2 == 23",
			expected: "( x2 == 23 )",
			hasError: false,
		},
		{
			test:     "var_long == 23",
			expected: "( var_long == 23 )",
			hasError: false,
		},
		{
			test:     "cpu == 23%",
			expected: "( cpu == 23 % )",
			hasError: false,
		},
		{
			test:     "cpu == 23% || x < 2",
			expected: "( ( cpu == 23 % ) || ( x < 2 ) )",
			hasError: false,
		},
		{
			test:     "cpu == 23% || x < 2 || y == 2",
			expected: "( ( ( cpu == 23 % ) || ( x < 2 ) ) || ( y == 2 ) )",
			hasError: false,
		},
		{
			test:     "x == 2 && cpu == 23% || x < 2",
			expected: "( ( ( x == 2 ) && ( cpu == 23 % ) ) || ( x < 2 ) )",
			hasError: false,
		},
		{
			test:     "cpu == 23% && mem >= 20Gib",
			expected: "( ( cpu == 23 % ) && ( mem >= 20 Gib ) )",
			hasError: false,
		},
		{
			test:     "cpu < 23% && mem > 20Gib",
			expected: "( ( cpu < 23 % ) && ( mem > 20 Gib ) )",
			hasError: false,
		},
		{
			test:     "(cpu == 23% && mem >= 20Gib) && x<=20",
			expected: "( ( ( cpu == 23 % ) && ( mem >= 20 Gib ) ) && ( x <= 20 ) )",
			hasError: false,
		},
		{
			test:     " x == 2 && ( x == 2 && ( x == 2 && y == 0 ) )",
			expected: "( ( x == 2 ) && ( ( x == 2 ) && ( ( x == 2 ) && ( y == 0 ) ) ) )",
			hasError: false,
		},
		{
			test:     " x == 2 || ( x == 2 || ( x == 2 || y == 0 ) )",
			expected: "( ( x == 2 ) || ( ( x == 2 ) || ( ( x == 2 ) || ( y == 0 ) ) ) )",
			hasError: false,
		},
		{
			test:     "(cpu == 23% && mem >= 20Gib) || (x<=20 && y == 20%)",
			expected: "( ( ( cpu == 23 % ) && ( mem >= 20 Gib ) ) || ( ( x <= 20 ) && ( y == 20 % ) ) )",
			hasError: false,
		},
		{
			test:     "name == 2  description != 2",
			hasError: true,
		},
		{
			test:     "name == 2 &&",
			hasError: true,
		},
		{
			test:     "2 > name",
			hasError: true,
		},
		{
			test:     "&& name > 2",
			hasError: true,
		},
		{
			test:     "2 > 2",
			expected: "( 2 > 2 )",
			hasError: false,
		},
		{
			test:     "x != nil",
			expected: "( x != nil )",
			hasError: false,
		},
	}

	for idx, data := range exprs {
		t.Run(fmt.Sprintf("test%d: %s", idx+1, data.test), func(t *testing.T) {
			searchExpr, err := parse([]byte(data.test))
			if err != nil && !data.hasError {
				t.Errorf("parse error: %v", err)
				return
			}

			if data.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Equal(t, data.expected, searchExpr.String())
			}

		})
	}
}
