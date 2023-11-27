package rcj

import (
	"context"
	"math"

	"github.com/sirupsen/logrus"

	"golang.org/x/exp/slices"
)

type NodeType string

const (
	CheckNode NodeType = "CheckNode"
	VarNode   NodeType = "VarNode"
)

type Graph struct {
	Nodes   []*Node
	NodeMap map[NodeType]map[int]*Node
	Loops   int
}

func (g *Graph) CreateNode(t NodeType) *Node {

	i := len(g.NodeMap[CheckNode])
	if t != CheckNode {
		i = len(g.NodeMap[VarNode])
	}

	n := &Node{
		Connections: make([]int, 0),
		Index:       i,
		Type:        t,
	}

	g.Nodes = append(g.Nodes, n)
	g.NodeMap[t][i] = n

	return n
}

func (g *Graph) RemoveNode(index int, nodeType NodeType) bool {
	i := slices.IndexFunc(g.Nodes, func(n *Node) bool {
		return n.Index == index && n.Type == nodeType
	})
	if i == -1 {
		return false
	}

	n := g.Nodes[i]
	otherNodes := g.NodeMap[CheckNode]
	if nodeType == CheckNode {
		otherNodes = g.NodeMap[VarNode]
	}

	for _, c := range n.Connections {
		n.DisconnectFrom(otherNodes[c])
	}

	g.Nodes[i] = g.Nodes[len(g.Nodes)-1]
	g.Nodes = g.Nodes[:len(g.Nodes)-1]

	delete(g.NodeMap[nodeType], index)

	return true
}

type Node struct {
	Connections []int
	Index       int
	Type        NodeType
	Weight      int
	Loop        int
}

func (n *Node) ConnectTo(n2 *Node) {
	if n.Type == n2.Type {
		panic("same type not allowed")
	}

	i := slices.IndexFunc(n.Connections, func(index int) bool {
		return n2.Index == index
	})
	if 0 <= i {
		return
	}

	n.Connections = append(n.Connections, n2.Index)
	n2.Connections = append(n2.Connections, n.Index)
}

func (n *Node) DisconnectFrom(n2 *Node) {
	if n.Type == n2.Type {
		panic("same type not allowed")
	}

	i := slices.IndexFunc(n.Connections, func(index int) bool {
		return n2.Index == index
	})
	if i == -1 {
		return
	}

	n.Connections[i] = n.Connections[len(n.Connections)-1]
	n.Connections = n.Connections[:len(n.Connections)-1]

	i = slices.IndexFunc(n2.Connections, func(index int) bool {
		return n.Index == index
	})
	if i == -1 {
		return
	}

	n2.Connections[i] = n2.Connections[len(n2.Connections)-1]
	n2.Connections = n2.Connections[:len(n2.Connections)-1]
}

func Build(ctx context.Context, girth, count int) *Graph {
	logrus.Infof("Building RCJ(%v,%v)", girth, count)
	g := &Graph{
		Nodes:   make([]*Node, 0),
		NodeMap: map[NodeType]map[int]*Node{CheckNode: {}, VarNode: {}},
	}

	currentGraph := [][]*Node{makeLoop(girth, g)}

	for len(currentGraph) < count/2 {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		//we make the graphCopy
		// we try and make it equal in size to the currentGraph

		clearWeights(ctx, currentGraph)
		updateWeights(ctx, girth, currentGraph, g)

		graphCopy := copyGraph(ctx, currentGraph, g)
		connectGraphs(ctx, currentGraph, graphCopy, g)

		currentGraph = append(currentGraph, graphCopy...)
	}
	return g
}

func copyGraph(ctx context.Context, currentGraph [][]*Node, g *Graph) [][]*Node {
	//we assume all the varNodes currently in g are for this currentGraph
	vars := make(map[int][]int)
	oldToNewVars := make(map[int]*Node)
	for _, n := range g.NodeMap[VarNode] {
		vars[n.Index] = n.Connections
	}

	newChecks := make(map[int]*Node)
	minCheckIndex := math.MaxInt
	checksNeeded := len(g.NodeMap[CheckNode])
	for i := 0; i < checksNeeded; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		c := g.CreateNode(CheckNode)
		newChecks[c.Index] = c
		if c.Index < minCheckIndex {
			minCheckIndex = c.Index
		}
	}

	for oldVar, conns := range vars {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		v := g.CreateNode(VarNode)

		for _, c := range conns {
			newChecks[c+minCheckIndex].ConnectTo(v)
		}

		oldToNewVars[oldVar] = v
	}

	// now construct the loops the hard way sigh
	loops := make([][]*Node, 0)

	for _, loop := range currentGraph {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		l := make([]*Node, len(loop))
		for j, n := range loop {
			if n.Type == CheckNode {
				l[j] = newChecks[n.Index+minCheckIndex]
			} else {
				l[j] = oldToNewVars[n.Index]
			}
			l[j].Loop = g.Loops + 1
		}
		g.Loops++
		loops = append(loops, loop)
	}

	return loops
}

func clearWeights(ctx context.Context, currentGraph [][]*Node) {
	for _, loop := range currentGraph {
		select {
		case <-ctx.Done():
			return
		default:
		}

		for _, n := range loop {
			n.Weight = math.MaxInt
		}
	}
}
func updateWeights(ctx context.Context, girth int, currentGraph [][]*Node, g *Graph) {
	// here we pick the smallest index with the fewest connections
	loop := currentGraph[0]
	currentNode := loop[0]
	conns := len(loop[0].Connections)

	for _, n := range loop {
		if n.Type == CheckNode && len(n.Connections) < conns {
			currentNode = n
		}
	}

	// now with the starting node we run the update alg
	currentNode.Weight = 1
	possibleConnections := make(map[int]*Node) // index/node for checkNodes only

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		rollOut(ctx, girth, currentNode, possibleConnections, g)

		if len(possibleConnections) == 0 {
			break // nothing left to do
		}

		for _, n := range possibleConnections {
			currentNode = n
			break
		}
		delete(possibleConnections, currentNode.Index)
		currentNode.Weight = 1
	}
}

func rollOut(ctx context.Context, girth int, node *Node, possibleConnections map[int]*Node, g *Graph) {
	// starting at node, we do a bread search, updating weights
	// along the way. we stop when all paths have stopped on a checkNode
	// with weight>= girth/2-2

	// when we come to a node with a nonzero weight we update and continue advancing provided
	// the current weight is not <= to the new weight to be assigned

	maxWeight := girth/2 - 2
	currWeight := node.Weight + 1
	currType := VarNode
	currIndices := node.Connections

	for len(currIndices) > 0 {

		nextIndices := make([]int, 0)

		for _, c := range currIndices {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n := g.NodeMap[currType][c]
			if n.Weight <= currWeight {
				continue
			}

			if currType == CheckNode &&
				n.Weight >= maxWeight &&
				n.Weight != math.MaxInt {
				// it might be in the possibleConnections so we'll removed it for now
				delete(possibleConnections, n.Index)
			}

			n.Weight = currWeight // update the weight

			if currType == CheckNode && n.Weight >= maxWeight {
				// we stop here because this is a possible Connections node
				possibleConnections[n.Index] = n
				continue
			}

			//else we want to get to the next set of nodes
			nextIndices = append(nextIndices, n.Connections...)
		}

		// setup for next round
		currIndices = nextIndices
		if currType == CheckNode {
			currType = VarNode
		} else {
			currType = CheckNode
		}
		currWeight++
	}
}

func connectGraphs(ctx context.Context, a, b [][]*Node, g *Graph) {
	// we look at graph a
	// and find nodes with weight 1 and connect the
	// corresponding node in graph b
	// they are both expected to be the same size
	for i, loop := range a {
		for j, n := range loop {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if n.Weight == 1 {
				v := g.CreateNode(VarNode)
				n.ConnectTo(v)
				b[i][j].ConnectTo(v)
			}
		}
	}
}

func makeLoop(girth int, g *Graph) []*Node {
	loop := make([]*Node, girth)
	for i := range loop {
		t := CheckNode
		if i%2 == 1 {
			t = VarNode
		}
		loop[i] = g.CreateNode(t)
		loop[i].Loop = g.Loops + 1
	}

	for i := range loop {
		loop[i].ConnectTo(loop[(i+1)%len(loop)])
	}
	g.Loops++
	return loop
}
