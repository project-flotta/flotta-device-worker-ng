package scheduler

import (
	"testing"
	"time"

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

	input <- "done"
	<-time.After(1 * time.Second)

	result, err = future.Poll()
	g.Expect(result.IsReady()).To(Equal(true))
	g.Expect(result.Value).To(Equal("done"))
	g.Expect(err).To(BeNil())

	result, err = future.Poll()
	g.Expect(result.IsReady()).To(Equal(false))
	g.Expect(err).To(BeNil())

	input <- "second value"
	close(input)
	<-time.After(1 * time.Second)
	result, err = future.Poll()
	g.Expect(result.IsReady()).To(Equal(true))
	g.Expect(result.Value).To(Equal("second value"))
	g.Expect(future.Resolved()).To(BeTrue())
	g.Expect(err).To(BeNil())

	result, err = future.Poll()
	g.Expect(err).ToNot(BeNil())

}
