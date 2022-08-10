package containers

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
)

type element int

func (e element) ID() string {
	return fmt.Sprintf("%d", e)
}

func TestStore(t *testing.T) {
	g := NewWithT(t)
	store := NewStore[element]()

	store.Add(element(1))
	store.Add(element(2))
	store.Add(element(3))

	g.Expect(store.Len()).To(Equal(3))

	iter := store.Iter()
	for i := 1; i < 4; i++ {
		v, ok := iter.Next()
		g.Expect(ok).To(BeTrue())
		g.Expect(v).To(Equal(element(i)))
	}

	_, ko := iter.Next()
	g.Expect(ko).To(BeFalse())

	// delete 2
	store.Delete(element(2))
	g.Expect(store.Len()).To(Equal(2))

	iter = store.Iter()
	v, ok := iter.Next()
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(element(1)))

	v, ok = iter.Next()
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(element(3)))

	// delete 1
	store.Delete(element(1))
	g.Expect(store.Len()).To(Equal(1))
	iter = store.Iter()
	v, ok = iter.Next()
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(element(3)))

	store.Delete(element(3))
	g.Expect(store.Len()).To(Equal(0))

	store.Delete(element(4))
	g.Expect(store.Len()).To(Equal(0))
}
