package scheduler

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
)

type element int

func (e element) ID() string {
	return fmt.Sprintf("%d", e)
}

func (e element) Name() string {
	return fmt.Sprintf("%d", e)
}

func TestStore(t *testing.T) {
	g := NewWithT(t)
	store := NewStore[element]()

	store.Add(element(1))
	store.Add(element(2))
	store.Add(element(3))

	g.Expect(store.Len()).To(Equal(3))

	// find 2
	_, ok := store.FindByName("2")
	g.Expect(ok).To(BeTrue())

	// delete 2
	store.Delete(element(2))
	g.Expect(store.Len()).To(Equal(2))

	// delete 1
	store.Delete(element(1))
	g.Expect(store.Len()).To(Equal(1))

	store.Delete(element(3))
	g.Expect(store.Len()).To(Equal(0))

	store.Delete(element(4))
	g.Expect(store.Len()).To(Equal(0))
}
