package interpreter

type lexer struct {
	src     []byte
	ch      byte
	offset  int
	pos     int
	nextPos int
}

func newLexer(src []byte) *lexer {
	l := &lexer{src: src}
	l.next()

	return l
}

func (l *lexer) Scan() (int, Token, string) {
	for l.ch == ' ' || l.ch == '\t' {
		l.next()
	}

	if l.ch == 0 {
		return l.pos, EOL, ""
	}

	tok := ILLEGAL
	pos := l.pos
	val := ""

	ch := l.ch
	l.next()

	// keywords
	if isAlpha(ch) || isSymbol(ch) {
		start := l.offset - 2
		for isAlpha(l.ch) || isDigit(l.ch) || isSymbol(l.ch) {
			l.next()
		}
		name := string(l.src[start : l.offset-1])
		tok := STRING
		val = name

		return pos, tok, val
	}

	switch ch {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		chars := make([]byte, 0, 32)
		chars = append(chars, ch)
		countDot := 0
		for isDigit(l.ch) || l.ch == '.' {
			if l.ch == '.' {
				countDot += 1
			}
			c := l.ch
			l.next()
			chars = append(chars, c)
		}
		tok = NUMBER
		val = string(chars)
		// allow only one dot in the number
		if countDot > 1 {
			tok = ILLEGAL
		}
	case '(':
		tok = LPAREN
	case ')':
		tok = RPAREN
	case '-':
		tok = DEC
	case '!':
		switch l.ch {
		case '=':
			tok = NOT_EQUALS
			l.next()
		default:
			tok = ILLEGAL
		}
	case '=':
		switch l.ch {
		case '=':
			tok = EQUALS
			l.next()
		default:
			tok = ILLEGAL
		}
	case '<':
		switch l.ch {
		case '=':
			tok = LTE
			l.next()
		default:
			tok = LESS
		}
	case '>':
		switch l.ch {
		case '=':
			tok = GTE
			l.next()
		default:
			tok = GREATER
		}
	case '&':
		switch l.ch {
		case '&':
			tok = AND
			l.next()
		default:
			tok = ILLEGAL
		}
	case '|':
		switch l.ch {
		case '|':
			tok = OR
			l.next()
		default:
			tok = ILLEGAL
		}
	default:
		tok = ILLEGAL
		val = "unexpected char"
	}

	return l.pos, tok, val

}

// Load the next character into l.ch (or 0 on end of input) and update line position.
func (l *lexer) next() {
	l.pos = l.nextPos
	if l.offset >= len(l.src) {
		// For last character, move offset 1 past the end as it
		// simplifies offset calculations in NAME and NUMBER
		if l.ch != 0 {
			l.ch = 0
			l.offset++
			l.nextPos++
		}
		return
	}
	ch := l.src[l.offset]
	l.ch = ch
	l.nextPos++
	l.offset++
}

func isAlpha(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isSymbol(ch byte) bool {
	return ch == '_' || ch == '%'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
