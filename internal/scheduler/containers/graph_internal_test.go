package containers

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
)

/*
*   ready -> deploying -> deployed -> running
*					   \
*                       -> deployed2 -> running
 */
func TestGraph(t *testing.T) {
	e := NewWithT(t)
	g := NewGraph[int, string]()

	ready := g.CreateNode("ready")
	deploying := g.CreateNode("deploying")
	deployed := g.CreateNode("deployed")
	deployed2 := g.CreateNode("deployed2")
	running := g.CreateNode("running")
	stopped := g.CreateNode("stopped")
	g.AddEdge(ready, deploying, 1)
	g.AddEdge(deploying, deployed, 2)
	g.AddEdge(deployed, running, 3)

	g.AddEdge(deployed, stopped, 3)

	g.AddEdge(deploying, deployed2, 2)
	g.AddEdge(deployed2, running, 3)

	path, err := g.FindPath("ready", "running")
	e.Expect(err).To(BeNil())
	e.Expect(len(path)).To(Equal(2))
}

/*
*                                               exit
*                                             /
*   ready -- deploying -- deployed -- running -- stopped
*					   \              /
*                       -- deployed2 /
 */
func TestGraph2(t *testing.T) {
	e := NewWithT(t)
	g := NewGraph[int, string]()
	ready := g.CreateNode("ready")
	deploying := g.CreateNode("deploying")
	deployed := g.CreateNode("deployed")
	deployed2 := g.CreateNode("deployed2")
	running := g.CreateNode("running")
	stopped := g.CreateNode("stopped")
	exited := g.CreateNode("exit")

	g.AddEdge(ready, deploying, 1)
	g.AddEdge(deploying, deployed, 2)
	g.AddEdge(deploying, deployed2, 2)

	g.AddEdge(deployed, running, 3)
	g.AddEdge(deployed2, running, 3)

	g.AddEdge(running, exited, 4)
	g.AddEdge(running, stopped, 4)

	path, err := g.FindPath("ready", "exit")
	e.Expect(err).To(BeNil())
	e.Expect(len(path)).To(Equal(2))
	fmt.Printf("%+v\n", path)
}

func TestGraph3(t *testing.T) {
	e := NewWithT(t)
	g := NewGraph[int, string]()
	ready := g.CreateNode("ready")
	deploying := g.CreateNode("deploying")

	g.AddEdge(ready, deploying, 1)
	path, err := g.FindPath("ready", "ready")
	e.Expect(err).To(BeNil())
	fmt.Printf("%+v\n", path)
	e.Expect(len(path)).To(Equal(1))
}

func TestPodmanTask(t *testing.T) {
	e := NewWithT(t)
	g := NewGraph[int, string]()

	ready := g.CreateNode("ready")

	// root -- markBasedType --> deploying
	deployingNode := g.CreateNode("deploying")
	g.AddEdge(ready, deployingNode, 1)

	// deploying -- eventBasedType -- > deployed
	deployedNode := g.CreateNode("deployed")
	g.AddEdge(deployingNode, deployedNode, 2)

	// deploying -- eventBasedType -- > errored
	// errored -- markBasedType --> ready
	erroredNode := g.CreateNode("error")
	g.AddEdge(deployingNode, erroredNode, 2)
	g.AddEdge(erroredNode, ready, 1)

	// deployed -- markBasedType --> running
	runningNode := g.CreateNode("running")
	g.AddEdge(deployedNode, runningNode, 2)

	// deployed -- eventBasedType --> degraded
	// running -- eventBasedType --> running
	degradedNode := g.CreateNode("degraded")
	g.AddEdge(deployedNode, degradedNode, 2)
	g.AddEdge(runningNode, degradedNode, 2)

	// deployed -- eventBasedType --> exited
	// running -- eventBasedType --> exited
	// exited -- markBasedType --> ready
	exitNode := g.CreateNode("exited")
	g.AddEdge(deployedNode, exitNode, 2)
	g.AddEdge(runningNode, exitNode, 2)
	g.AddEdge(exitNode, ready, 1)

	// running -- markBasedType --> stoppingNode
	// degradedNode -- markBasedType --> stoppingNode
	stoppingNode := g.CreateNode("stopping")
	g.AddEdge(runningNode, stoppingNode, 1)
	g.AddEdge(degradedNode, stoppingNode, 1)

	// stoppingNode -- eventBasedType --> stoppedNode
	// stoppedNode -- markBasedType --> ready
	stoppedNode := g.CreateNode("stopped")
	g.AddEdge(stoppingNode, stoppedNode, 2)
	g.AddEdge(stoppedNode, ready, 1)

	// stoppedNode -- markBasedType --> inactiveNode
	// exitNode -- markBasedType --> inactiveNode
	// inactiveNode -- markBasedType --> ready
	inactiveNode := g.CreateNode("inactive")
	g.AddEdge(stoppedNode, inactiveNode, 1)
	g.AddEdge(exitNode, inactiveNode, 1)
	g.AddEdge(inactiveNode, ready, 1)

	// exitNode -- markBasedType --> delete
	// stopped -- markBasedType --> delete
	// inactiveNode -- markBasedType --> delete
	// ready -- markBasedType --> delete
	deleteNode := g.CreateNode("deletion")
	g.AddEdge(exitNode, deleteNode, 1)
	g.AddEdge(stoppedNode, deleteNode, 1)
	g.AddEdge(inactiveNode, deleteNode, 1)
	g.AddEdge(ready, deleteNode, 1)

	path, err := g.FindPath("ready", "deletion")
	e.Expect(err).To(BeNil())
	for _, p := range path {
		fmt.Printf("%+v\n", p)
	}
}
