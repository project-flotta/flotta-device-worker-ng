package task

import (
	"github.com/tupyy/device-worker-ng/internal/scheduler/containers"
)

type blueprintType int

func (b blueprintType) String() string {
	switch b {
	case podmanBlueprintType:
		return "podman"
	case cronJobBlueprintType:
		return "cron"
	default:
		return "unknown"
	}
}

const (
	podmanBlueprintType blueprintType = iota
	cronJobBlueprintType
)

type EdgeType int

const (
	// EventBasedEdgeType is a type of edge which can be crossed following an event
	EventBasedEdgeType EdgeType = iota
	// MarkBasedEdgeType is a type of edge which can be crossed following a mark
	MarkBasedEdgeType
)

type PathElement[S, T any] struct {
	Node S
	Edge T
}

type Path []PathElement[State, EdgeType]
type Paths []Path

type blueprint struct {
	Kind  blueprintType
	graph *containers.Graph[EdgeType, State]
}

func newPodmanBlueprint() *blueprint {
	return &blueprint{
		graph: createPodmanGraph(),
		Kind:  podmanBlueprintType,
	}
}

func (b *blueprint) FindPath(start, end State) (Paths, error) {
	res, err := b.graph.FindPath(start, end)
	if err != nil {
		return nil, err
	}
	paths := make([]Path, 0, len(res))
	for _, r := range res {
		path := make([]PathElement[State, EdgeType], 0, len(r))
		for _, val := range r {
			for k, v := range val {
				path = append(path, PathElement[State, EdgeType]{
					Node: k,
					Edge: v,
				})
				break
			}
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func createPodmanGraph() *containers.Graph[EdgeType, State] {
	g := containers.NewGraph[EdgeType, State]()

	readyNode := g.CreateNode(ReadyState)
	// root -- markBasedType --> deploying
	deployingNode := g.CreateNode(DeployingState)
	g.AddEdge(readyNode, deployingNode, MarkBasedEdgeType)

	// deploying -- eventBasedType -- > deployed
	deployedNode := g.CreateNode(DeployedState)
	g.AddEdge(deployingNode, deployedNode, EventBasedEdgeType)

	// deploying -- eventBasedType -- > errored
	// errored -- markBasedType --> ready
	erroredNode := g.CreateNode(ErrorState)
	g.AddEdge(deployingNode, erroredNode, EventBasedEdgeType)

	// deployed -- markBasedType --> running
	runningNode := g.CreateNode(RunningState)
	g.AddEdge(deployedNode, runningNode, EventBasedEdgeType)

	// deployed -- eventBasedType --> degraded
	// running -- eventBasedType --> running
	// deployed -- eventBasedType --> running
	degradedNode := g.CreateNode(DegradedState)
	g.AddEdge(deployedNode, degradedNode, EventBasedEdgeType)
	g.AddEdge(runningNode, degradedNode, EventBasedEdgeType)
	g.AddEdge(degradedNode, runningNode, EventBasedEdgeType)

	// deployed -- eventBasedType --> exited
	// running -- eventBasedType --> exited
	// exited -- eventBasedType --> running
	// exited -- markBasedType --> ready
	exitNode := g.CreateNode(ExitedState)
	g.AddEdge(deployedNode, exitNode, EventBasedEdgeType)
	g.AddEdge(runningNode, exitNode, EventBasedEdgeType)
	g.AddEdge(exitNode, runningNode, EventBasedEdgeType)

	// running -- markBasedType --> stoppingNode
	// degradedNode -- markBasedType --> stoppingNode
	stoppingNode := g.CreateNode(StoppingState)
	g.AddEdge(runningNode, stoppingNode, MarkBasedEdgeType)
	g.AddEdge(degradedNode, stoppingNode, MarkBasedEdgeType)
	g.AddEdge(deployedNode, stoppingNode, MarkBasedEdgeType)

	// stoppingNode -- eventBasedType --> stoppedNode
	// stoppedNode -- markBasedType --> ready
	// ready -> markBasedType -> stopped
	stoppedNode := g.CreateNode(StoppedState)
	g.AddEdge(stoppingNode, stoppedNode, EventBasedEdgeType)
	g.AddEdge(readyNode, stoppedNode, MarkBasedEdgeType)

	// Inactive is the end point. There is no natural exit from this node.
	// In order to get the task back to ready, the task must be marked with 'active'
	// stoppedNode -- markBasedType --> inactiveNode
	// exitNode -- markBasedType --> inactiveNode
	// readyNode -- markBasedType --> inactive
	inactiveNode := g.CreateNode(InactiveState)
	g.AddEdge(stoppedNode, inactiveNode, MarkBasedEdgeType)
	g.AddEdge(exitNode, inactiveNode, MarkBasedEdgeType)
	g.AddEdge(readyNode, inactiveNode, MarkBasedEdgeType)

	// exitNode -- markBasedType --> delete
	// stopped -- markBasedType --> delete
	// inactiveNode -- markBasedType --> delete
	// ready -- markBasedType --> delete
	deleteNode := g.CreateNode(DeletionState)
	g.AddEdge(exitNode, deleteNode, MarkBasedEdgeType)
	g.AddEdge(stoppedNode, deleteNode, MarkBasedEdgeType)
	g.AddEdge(inactiveNode, deleteNode, MarkBasedEdgeType)
	g.AddEdge(readyNode, deleteNode, MarkBasedEdgeType)

	return g
}
