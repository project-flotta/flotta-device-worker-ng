package interpreter

type valueType uint8

const (
	typeNull valueType = iota
	typeBool
	typeNum
)

// An generic value (these are passed around by value)
type value struct {
	typ valueType // Type of value
	b   bool      // Bool value
	n   float32   // Numeric value (for typeNum)
}

// Create a new number value
func num(n float32) value {
	return value{typ: typeNum, n: n}
}

// Create a numeric value from a Go bool
func boolean(b bool) value {
	return value{typ: typeBool, b: b}
}

func null() value {
	return value{typ: typeNull}
}
