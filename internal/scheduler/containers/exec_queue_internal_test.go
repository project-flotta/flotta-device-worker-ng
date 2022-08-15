package containers

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestExecQueue(t *testing.T) {
	g := NewWithT(t)
	q := ExecutionQueue[int, int]{}

	q.Push(1, 1)
	q.Push(2, 2)
	q.Push(1, 3)
	q.Push(2, 4)
	g.Expect(q.Size()).To(Equal(4))

	q.Sort(1)

	a1, v1, e1 := q.Pop()
	a2, v2, e2 := q.Pop()
	g.Expect(a1).To(Equal(1))
	g.Expect(a2).To(Equal(1))
	g.Expect(v1).To(Equal(1))
	g.Expect(v2).To(Equal(3))
	g.Expect(e1).To(BeNil())
	g.Expect(e2).To(BeNil())

	g.Expect(q.Size()).To(Equal(2))
}

func TestExecQueue2(t *testing.T) {
	g := NewWithT(t)
	q := ExecutionQueue[int, int]{}

	q.Push(1, 1)
	q.Push(2, 2)
	_, v1, _ := q.Pop()
	_, v2, _ := q.Pop()
	_, _, e3 := q.Pop()
	g.Expect(v1).To(Equal(1))
	g.Expect(v2).To(Equal(2))
	g.Expect(e3).ToNot(BeNil())

	q.Push(1, 1)
	g.Expect(q.Size()).To(Equal(1))
}
