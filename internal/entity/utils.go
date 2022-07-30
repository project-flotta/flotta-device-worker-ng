package entity

type Option[T any] struct {
	Value T
	None  bool
}
