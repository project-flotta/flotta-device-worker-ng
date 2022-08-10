package containers

import (
	"errors"
	"fmt"
	"strings"
)

type node[S comparable, T any] struct {
	next   *node[S, T]
	prev   *node[S, T]
	action S
	value  T
}

func NewQueue[S comparable, T any]() *Queue[S, T] {
	return &Queue[S, T]{}
}

type Queue[S comparable, T any] struct {
	root *node[S, T]
	size int
}

func (s *Queue[S, T]) Push(a S, v T) {
	if s.root == nil {
		s.root = &node[S, T]{action: a, value: v}
		s.size++

		return
	}

	s.pushEnd(s.root, a, v)
}

func (s *Queue[S, T]) Pop() (S, T, error) {
	var (
		result T
		action S
	)
	if s.root == nil {
		return action, result, errors.New("queue empty")
	}

	root := s.root
	root.prev = nil

	if s.root.next != nil {
		s.root = s.root.next
		s.root.prev = nil
	} else {
		s.root = nil
	}

	s.size--

	return root.action, root.value, nil
}

func (s *Queue[S, T]) Size() int {
	return s.size
}

// Sort sorts(bubble sort) the queue by action meaning it will put all the nodes
// with action equal to 'action' at the beginning of the queue
func (s *Queue[S, T]) Sort(action S) {
	if s.size < 2 {
		return
	}

	for {
		n := s.root
		dirty := false
		for n.next != nil {
			next := n.next
			if n.action != action && next.action == action {
				s.swap(n, next)
				dirty = true
				break
			}
			n = next
		}

		if !dirty {
			break
		}
	}
}

func (s *Queue[S, T]) swap(n1, n2 *node[S, T]) {
	if n1.prev == nil && n2.next == nil {
		n2.next = n1
		n1.prev = n2
		return
	}

	if n1.prev != nil && n2.next == nil {
		left := n1.prev

		n2.prev = left
		left.next = n2
		n2.next = n1
		n1.prev = n2
		n1.next = nil
		return
	}

	if n1.prev == nil && n2.next != nil {
		right := n2.next

		right.prev = n1
		n1.next = right
		n1.prev = n2

		n2.prev = nil
		n2.next = n1
		s.root = n2
		return

	}

	if n1.prev != nil && n2.next != nil {
		left := n1.prev
		right := n2.next

		left.next = n2
		n2.prev = left
		n2.next = n1
		n1.prev = n2
		n1.next = right
		right.prev = n1
		return
	}
}

func (s *Queue[S, T]) String() string {
	var sb strings.Builder
	n := s.root
	for n != nil {
		fmt.Fprintf(&sb, "[%v]%v->", n.action, n.value)
		n = n.next
	}

	return sb.String()
}

func (s *Queue[S, T]) pushEnd(n *node[S, T], a S, v T) {
	if n.next == nil {
		newNode := &node[S, T]{value: v, action: a}
		n.next = newNode
		newNode.prev = n
		s.size++

		return
	}

	s.pushEnd(n.next, a, v)
}
