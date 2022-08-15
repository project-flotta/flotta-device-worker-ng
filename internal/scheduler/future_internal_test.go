package scheduler

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestFuture(t *testing.T) {
	g := NewWithT(t)

	input := make(chan string)

	future := NewFuture(input)

	// poll twice
	result, err := future.Poll()
	g.Expect(result.IsPending()).To(Equal(true))
	g.Expect(err).To(BeNil())

	result, err = future.Poll()
	g.Expect(result.IsPending()).To(Equal(true))
	g.Expect(err).To(BeNil())

	input <- "msg"
	input <- "msg2"
	input <- "msg3"

	r, err := future.Poll()
	g.Expect(r.Value).To(Equal("msg"))
	g.Expect(err).To(BeNil())

	r, err = future.Poll()
	g.Expect(r.Value).To(Equal("msg2"))
	g.Expect(err).To(BeNil())

	r, err = future.Poll()
	g.Expect(r.Value).To(Equal("msg3"))
	g.Expect(err).To(BeNil())

	r, err = future.Poll()
	g.Expect(r.IsPending()).To(BeTrue())
}
