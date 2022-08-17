package scheduler

import "github.com/tupyy/device-worker-ng/internal/scheduler/containers"

type blueprintType int

const (
	podmanBlueprintType blueprintType = iota
	cronJobBlueprintType
)

type edgeType int

const (
	// eventBasedEdgeType is a type of edge which can be crossed following an event
	eventBasedEdgeType edgeType = iota
	// markBasedEdgeType is a type of edge which can be crossed following a mark
	markBasedEdgeType
)

type blueprint struct {
	Kind  blueprintType
	graph *containers.Graph[edgeType, TaskState]
}

func NewPodmanBlueprint() *blueprint {
	return &blueprint{
		graph: createPodmanGraph(),
		Kind:  podmanBlueprintType,
	}
}

func (b *blueprint) To(from TaskState) ([]*containers.Node[edgeType, TaskState], error) {
	return nil, nil
}

func createPodmanGraph() *containers.Graph[edgeType, TaskState] {
	g := containers.NewGraph[edgeType, TaskState]()

	readyNode := g.CreateNode(TaskStateReady)
	// root -- markBasedType --> deploying
	deployingNode := g.CreateNode(TaskStateDeploying)
	g.AddEdge(readyNode, deployingNode, markBasedEdgeType)

	// deploying -- eventBasedType -- > deployed
	deployedNode := g.CreateNode(TaskStateDeployed)
	g.AddEdge(deployingNode, deployedNode, eventBasedEdgeType)

	// deploying -- eventBasedType -- > errored
	// errored -- markBasedType --> ready
	erroredNode := g.CreateNode(TaskStateError)
	g.AddEdge(deployingNode, erroredNode, eventBasedEdgeType)
	g.AddEdge(erroredNode, readyNode, markBasedEdgeType)

	// deployed -- markBasedType --> running
	runningNode := g.CreateNode(TaskStateRunning)
	g.AddEdge(deployedNode, runningNode, eventBasedEdgeType)

	// deployed -- eventBasedType --> degraded
	// running -- eventBasedType --> running
	degradedNode := g.CreateNode(TaskStateDegraded)
	g.AddEdge(deployedNode, degradedNode, eventBasedEdgeType)
	g.AddEdge(runningNode, degradedNode, eventBasedEdgeType)

	// deployed -- eventBasedType --> exited
	// running -- eventBasedType --> exited
	// exited -- markBasedType --> ready
	exitNode := g.CreateNode(TaskStateExited)
	g.AddEdge(deployedNode, exitNode, eventBasedEdgeType)
	g.AddEdge(runningNode, exitNode, eventBasedEdgeType)
	g.AddEdge(exitNode, readyNode, markBasedEdgeType)

	// running -- markBasedType --> stoppingNode
	// degradedNode -- markBasedType --> stoppingNode
	stoppingNode := g.CreateNode(TaskStateStopping)
	g.AddEdge(runningNode, stoppingNode, markBasedEdgeType)
	g.AddEdge(degradedNode, stoppingNode, markBasedEdgeType)

	// stoppingNode -- eventBasedType --> stoppedNode
	// stoppedNode -- markBasedType --> ready
	stoppedNode := g.CreateNode(TaskStateStopped)
	g.AddEdge(stoppingNode, stoppedNode, eventBasedEdgeType)
	g.AddEdge(stoppedNode, readyNode, markBasedEdgeType)

	// stoppedNode -- markBasedType --> inactiveNode
	// exitNode -- markBasedType --> inactiveNode
	// inactiveNode -- markBasedType --> ready
	inactiveNode := g.CreateNode(TaskStateInactive)
	g.AddEdge(stoppedNode, inactiveNode, markBasedEdgeType)
	g.AddEdge(exitNode, inactiveNode, markBasedEdgeType)
	g.AddEdge(inactiveNode, readyNode, markBasedEdgeType)

	// exitNode -- markBasedType --> delete
	// stopped -- markBasedType --> delete
	// inactiveNode -- markBasedType --> delete
	// ready -- markBasedType --> delete
	deleteNode := g.CreateNode(TaskStateDeletion)
	g.AddEdge(exitNode, deleteNode, markBasedEdgeType)
	g.AddEdge(stoppedNode, deleteNode, markBasedEdgeType)
	g.AddEdge(inactiveNode, deleteNode, markBasedEdgeType)
	g.AddEdge(readyNode, deleteNode, markBasedEdgeType)

	return g
}
