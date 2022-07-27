package interpreter

import (
	"strings"
	"testing"
)

func testTokens(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{
			input:  "( ) == < <= > >= name 123 123.123 123.0 123. 1.2.3 name",
			output: "( ) == < <= > >= string number number number number illegal string EOL",
		},
		{
			input:  "cpu123 y0 variable",
			output: "string string string EOL",
		},
		{
			input:  "cpu == 23%",
			output: "string == number % EOL",
		},
		{
			input:  "cpu == 22% && mem <= 10Gib || network < 100kbs",
			output: "string == number % && string <= number string || string < number string EOL",
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			l := newLexer([]byte(test.input))

			tokens := []string{}
			for {
				_, tok, _ := l.Scan()
				tokens = append(tokens, tok.String())
				if tok == EOL {
					break
				}

			}

			output := strings.Join(tokens, " ")
			if strings.TrimSpace(output) != test.output {
				t.Errorf("expected %q, got %q", test.output, output)
			}
		})
	}
}
