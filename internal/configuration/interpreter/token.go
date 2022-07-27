package interpreter

type Token int

const (
	ILLEGAL Token = iota
	EOL

	// single character tokens
	LPAREN
	RPAREN
	PERCENT
	DEC

	// one or two character tokens
	AND
	EQUALS
	NOT_EQUALS
	DIV
	GTE
	GREATER
	LTE
	LESS
	OR

	// literals
	STRING
	NUMBER
)

var tokenNames = map[Token]string{
	ILLEGAL:    "illegal",
	EOL:        "EOL",
	LPAREN:     "(",
	RPAREN:     ")",
	PERCENT:    "%",
	DEC:        "-",
	AND:        "&&",
	DIV:        "/",
	EQUALS:     "==",
	NOT_EQUALS: "!=",
	GTE:        ">=",
	GREATER:    ">",
	LTE:        "<=",
	LESS:       "<",
	OR:         "||",
	STRING:     "string",
	NUMBER:     "number",
}

func (t Token) String() string {
	return tokenNames[t]
}
