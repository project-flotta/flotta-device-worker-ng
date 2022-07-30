package entity

type Option[T any] struct {
	Value T
	None  bool
}

type Result[T any] struct {
	Value T
	Error error
}
