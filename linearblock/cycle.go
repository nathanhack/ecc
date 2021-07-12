package linearblock

import (
	"fmt"
	mat "github.com/nathanhack/sparsemat"
	"math"
	"runtime"
	"strings"
	"sync"
)

//Node is a structure containing the Index and state of whether it is a check node or a variable node
type Node struct {
	Index int
	Check bool
}

//Cycle is a slice of Nodes
type Cycle []Node

//String returns a standard rep of the cycle
func (c Cycle) String() string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for i, n := range c {
		if n.Check {
			sb.WriteString(fmt.Sprintf("c:%v", n.Index))
		} else {
			sb.WriteString(fmt.Sprintf("v:%v", n.Index))
		}
		if i < len(c)-1 {
			sb.WriteString(" ")
		}
	}
	sb.WriteString("]")
	return sb.String()
}

//Equal compares two cycles to see if they are equal. To be equal
// the first node must be equal but they could be in opposite orderVector.
func (c Cycle) Equal(c2 Cycle) bool {
	if len(c) != len(c2) {
		return false
	}

	if len(c) == 0 {
		return true
	}

	if c[0] != c2[0] {
		return false
	}

	t1 := c[1:]
	t2 := c2[1:]

	//we try forward direction first
	failed := false
	for i, n := range t1 {
		if t2[i] != n {
			failed = true
			break
		}
	}

	if !failed {
		return true
	}

	//else we check the opposite direction
	l := len(t1)
	for i, n := range t1 {
		if t2[l-1-i] != n {
			return false
		}
	}
	return true
}

//SmallestCycle non-deterministically returns a cycle from the set of smallest cycles.
func SmallestCycle(m mat.SparseMat, parallel bool) Cycle {
	rows, _ := m.Dims()
	wg := sync.WaitGroup{}
	threadSize := 1
	if parallel {
		threadSize = runtime.NumCPU()
	}
	limit := make(chan bool, threadSize)
	mux := sync.RWMutex{}
	var smallest []Node
	girth := -1
	for i := 0; i < rows; i++ {
		wg.Add(1)
		limit <- true
		go func(index int) {
			c := smallestCycle(m, index, girth-2)

			mux.Lock()
			if len(smallest) == 0 || (len(c) != 0 && len(smallest) > len(c)) {
				smallest = c
				girth = len(smallest)
			}
			mux.Unlock()
			wg.Done()
			<-limit
		}(i)
	}
	wg.Wait()
	return smallest
}

func smallestCycle(m mat.SparseMat, checkIndex int, minGirth int) []Node {
	if minGirth < 0 {
		minGirth = math.MaxInt64
	}
	//we make a history that will alternate between variable nodes and check nodes
	// as we extend to each new hop away from the checkIndex
	history := make([]map[int]girthNode, 0)
	rows, _ := m.Dims()

	//we prime the history
	hop := make(map[int]girthNode)
	for i := range m.Row(checkIndex).NonzeroMap() {
		hop[i] = girthNode{parentIndex: checkIndex}
	}
	//if there was only one variable node then there is no way
	// this will have a loop
	if len(hop) == 1 {
		return nil
	}
	history = append(history, hop)

	for level := 1; level < 2*rows && level < minGirth/2+1; level++ {
		if level%2 == 0 {
			//this round we look at adding check nodes
			prevHop := history[level-1]
			hop := make(map[int]girthNode)
			for v, gn := range prevHop {
				for i := range m.Row(v).NonzeroMap() {
					if i == gn.parentIndex {
						continue
					}
					_, has := hop[i]
					if has {
						//so we make two node lists both start from this ith index
						// but one will be what's already in history/hop
						// and the other will be from this new girthNode
						// at the end we'll concatenate the lists (while removing dup the starting node)

						//first we get the lists (it contains all nodes except the current one)
						a := path(history, hop[i].parentIndex, true)
						b := path(history, v, true)

						//we add the connection to the checknode in questions
						a = append([]Node{{
							Index: checkIndex,
							Check: true,
						}}, a...)

						// we add the node to one of them
						a = append(a, Node{Index: i, Check: false})

						//next we reverse the orderVector of b
						reverse(b)

						//now concatenate the two together
						return append(a, b...)
					}
					hop[i] = girthNode{parentIndex: v}
				}
			}
			history = append(history, hop)
		} else {
			//this round we look at adding variable nodes
			prevHop := history[level-1]
			hop := make(map[int]girthNode)
			for v, gn := range prevHop {
				for i := range m.Column(v).NonzeroMap() {
					if i == gn.parentIndex {
						continue
					}
					_, has := hop[i]
					if has || i == checkIndex {
						//so we make two node lists both start from this ith index
						// but one will be what's already in history/hop
						// and the other will be from this new girthNode
						// at the end we'll concatenate the lists (while removing dup the starting node)

						//first we get the lists (it contains all nodes except the current one)
						a := path(history, hop[i].parentIndex, false)
						b := path(history, v, false)

						//we add the connection to the checknode in questions
						a = append([]Node{{
							Index: checkIndex,
							Check: true,
						}}, a...)

						// we add the node to one of them
						if i != checkIndex {
							a = append(a, Node{Index: i, Check: true})
						}

						//next we reverse the orderVector of b
						reverse(b)

						//now concatenate the two together
						return append(a, b...)
					}
					hop[i] = girthNode{parentIndex: v}
				}
			}
			history = append(history, hop)
		}
	}
	return nil
}

func path(history []map[int]girthNode, index int, check bool) []Node {
	//we start from the last item in the history
	histLen := len(history)

	path := make([]Node, 0, histLen+1)

	for i := histLen - 1; i >= 0; i-- {
		n := Node{
			Index: index,
			Check: check,
		}

		path = append(path, n)
		index = history[i][index].parentIndex
		check = !check
	}
	reverse(path) // now we correct the orderVector
	return path
}

func reverse(nodes []Node) {
	for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
}
