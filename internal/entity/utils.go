package entity

type Option[T any] struct {
	Value T
	None  bool
}

type Result[T any] struct {
	Value T
	Error error
}

type Pair[S, T any] struct {
	Name  S
	Value T
}

type Tuple[S, T any] struct {
	Value1 S
	Value2 T
}
