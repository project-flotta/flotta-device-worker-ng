package containers

type n[T any] struct {
	prev  *n[T]
	value T
}

type Queue[T any] struct {
	root *n[T]
	size int
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{}
}

func (s *Queue[T]) Push(p T) {
	if s.root == nil {
		s.root = &n[T]{value: p}
		s.size++

		return
	}

	s.pushEnd(s.root, p)
}

func (s *Queue[T]) Peek() T {
	var none T
	if s.root == nil {
		return none
	}
	return s.root.value
}

func (s *Queue[T]) Pop() T {
	var none T
	if s.root == nil {
		return none
	}

	root := s.root
	s.root = s.root.prev
	s.size--

	return root.value
}

func (s *Queue[T]) Size() int {
	return s.size
}

func (s *Queue[T]) pushEnd(node *n[T], p T) {
	if node.prev == nil {
		node.prev = &n[T]{value: p}
		s.size++

		return
	}

	s.pushEnd(node.prev, p)
}
