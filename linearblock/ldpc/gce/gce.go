package gce

import (
	"context"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/nathanhack/errorcorrectingcodes/linearblock"
	"github.com/nathanhack/errorcorrectingcodes/linearblock/internal"
	mat "github.com/nathanhack/sparsemat"
	"github.com/sirupsen/logrus"
	"sort"
)

// Based on the paper Constructing LDPC Codes with Any Desired Girth
//    by Chaohui Gao, Sen Liu, Dong Jiang, and Lijun Chen

//Search attempts to find a GCE parity matrix for the given checkNodes, variableNodes and girth in the given number of iterations.
// Threads if zero will use all current CPUs in parallel. There are cases when force would need to be used and the user is notified through logrus info messages.
// Lastly when it takes more than one iteration, then checkpoints can but used to save progress instead of waiting until the end.
func Search(ctx context.Context, checkNodes, variableNodes, girth, iterations, threads int, force bool, checkpoint func(currentBest *linearblock.LinearBlock)) (*linearblock.LinearBlock, error) {
	if girth%2 == 1 || girth < 4 {
		return nil, fmt.Errorf("girth must be an even number and >=4")
	}
	x := girth / 2
	if x > checkNodes {
		return nil, fmt.Errorf("girth not possible with number of checkNodes")
	}
	if x > variableNodes {
		return nil, fmt.Errorf("girth not possible with number of variableNodes")
	}

	if variableNodes <= checkNodes {
		return nil, fmt.Errorf("can only create GCE's with more variable nodes than check nodes ")
	}

	var best *gceState = nil
	var bestCopy *gceState = nil
	var err error

iterLoop:
	for iter := 0; iter < iterations; iter++ {
		logrus.Debugf("Iterations: %v", iter)
		state := newGceState(checkNodes, variableNodes)
		err = gceLdpcRunner(ctx, state, girth)
		if err != nil {
			//here we'll have the state to compare it against the best
			// we might be here if it's a particularly hard ldpc to create
			logrus.Debugf("Iterations: %v failed with error: %v", iter, err)
		}

		select {
		case <-ctx.Done():
			break iterLoop
		default:
		}

		// let's see if we got something possibly interesting
		if best == nil || bestCopy == nil || bestCopy.Cmp(state) > 0 {
			tmpCopy := state.Copy()
			finished := state.Finished()
			//next we need to special cases need for "force"
			if !finished {
				if !state.cn.Exhausted() {
					logrus.Debugf("unable to exhaust checknodes, checknodes must be exhausted for succesful search")
					continue
				}
				if !force {
					logrus.Infof("failed to create a gce during %v iteration (consider using force set to true)", iter)
					continue
				}
				gceLdpcRunnerForce(ctx, state, girth)
			}

			// at this point we have a state that is "completed"
			order, g := internal.NewFromH(ctx, state.H, threads)
			if order == nil {
				continue
			}

			state.lb = &linearblock.LinearBlock{
				H: state.H,
				Processing: &linearblock.Systemic{
					HColumnOrder: order,
					G:            g,
				},
			}
			best = state
			bestCopy = tmpCopy
			
			if checkpoint != nil {
				checkpoint(best.lb)
			}

			// if it was a "best" solutions
			if finished {
				break iterLoop
			}
			continue
		}
		logrus.Debugf("Previous model was better.")
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("early termination")
	default:
	}

	if best == nil {
		return nil, fmt.Errorf("no LDPC found")
	}

	return best.lb, nil
}

func gceLdpcRunnerForce(ctx context.Context, state *gceState, girth int) {
	x := girth / 2

	logrus.Debugf("Force step 1 of 2")
	c1, v, success := findTwoNodes(state.H, state.cn.old, 2*x-1)
	for success {
		state.H.Set(c1, v, 1)
		c1, v, success = findTwoNodes(state.H, state.cn.old, 2*x-1)

		select {
		case <-ctx.Done():
			return
		default:
		}
	}

	// step 3
	logrus.Debugf("Force step 2 of 2")
	if !state.vn.Exhausted() {
		//we basically find places for all the vn's to dangle (not ideal)
		sort.Slice(state.cn.old, func(i, j int) bool {
			a := state.H.Row(i).HammingWeight()
			b := state.H.Row(j).HammingWeight()
			return a < b
		})

		// we want to limit having them all hang from one cn so we'll spread them across all
		// of them and loop back when we run out of check nodes
		for i := 0; i < state.vn.Len(); i++ {
			state.H.Set(state.cn.old[i%len(state.cn.old)], state.vn.new[i], 1)
			select {
			case <-ctx.Done():
				return
			default:
			}
		}

		//now update the internal state
		state.vn.PopAll()
	}
}

type gceState struct {
	H      mat.SparseMat
	cn, vn *Nodes
	lb     *linearblock.LinearBlock
}

func (s *gceState) Finished() bool {
	return s.cn.Exhausted() && s.vn.Exhausted()
}

func (s *gceState) Cmp(state *gceState) int {
	st := s.vn.Len() + s.cn.Len()
	statet := state.vn.Len() + state.cn.Len()

	switch {
	case st > statet:
		return 1
	case st < statet:
		return -1
	}
	return 0
}

func (s *gceState) Copy() *gceState {
	return &gceState{
		H:  mat.CSRMatCopy(s.H),
		cn: s.cn.Copy(),
		vn: s.vn.Copy(),
		lb: nil,
	}
}

func newGceState(checkNodes, variableNodes int) *gceState {
	return &gceState{
		H:  mat.DOKMat(checkNodes, variableNodes),
		cn: newNodes(checkNodes),
		vn: newNodes(variableNodes),
	}
}

func gceLdpcRunner(ctx context.Context, state *gceState, girth int) error {
	x := girth / 2

	//step 1: first we init the starting search by creating the first cycle
	logrus.Debugf("Step 1 of 4")
	initH(state, girth)

	logrus.Debugf("Step 2 of 4")
	// step 2
	hconst := x/2 - 1
	if x%2 == 1 {
		hconst = (x - 1) / 2
	}

	for !state.cn.Exhausted() {
		var dist, h int
		if state.cn.Len() >= hconst {
			h = hconst
			dist = x - (x % 2)
		} else {
			h = state.cn.Len()
			dist = 2 * (x - state.cn.Len() - 1)
		}
		c1, c2, success := findTwoNodes(state.H, state.cn.old, dist)

		if !success {
			//this occurs when there aren't "enough" cycles to provide opinions
			return fmt.Errorf("failed to find two check nodes to exhaust check nodes (%v), this can happen randomly but frequently happens when the matrix is too small for the requested girth", state.cn.Len())
		}
		//TODO handle the case when state.vn.Exhausted() has occured before state.cn.Exhausted()
		connect(c1, c2, h, state)
		select {
		case <-ctx.Done():
			return fmt.Errorf("early termination")
		default:
		}
	}

	// step 3
	logrus.Debugf("Step 3 of 4")
	bar := pb.New(state.vn.Len())
	for !state.vn.Exhausted() {
		if logrus.GetLevel() == logrus.DebugLevel {
			bar.Increment()
		}
		c1, c2, success := findTwoNodes(state.H, state.cn.old, 2*x-2)
		if !success {
			bar.Finish()
			return fmt.Errorf("failed to find two check nodes to exhaust variable nodes (%v), this can happen randomly but frequently happens when the matrix is too small for the requested girth", state.vn.Len())
		}
		connect(c1, c2, 0, state)
		select {
		case <-ctx.Done():
			return fmt.Errorf("early termination")
		default:
		}
	}
	bar.Finish()

	// step 4
	logrus.Debugf("Step 4 of 4")
	c1, v, success := findTwoNodes(state.H, state.cn.old, 2*x-1)
	bar = pb.New(0)
	for success {
		if logrus.GetLevel() == logrus.DebugLevel {
			bar.Increment()
		}
		state.H.Set(c1, v, 1)
		c1, v, success = findTwoNodes(state.H, state.cn.old, 2*x-1)
		select {
		case <-ctx.Done():
			return fmt.Errorf("early termination")
		default:
		}
	}
	bar.Finish()
	return nil
}

func connect(c1 int, c2 int, hconst int, state *gceState) {
	//we take hconst number of cn's and hconst+1 number of vn's to connect c1 and c2

	//we start with c1 and end with c2
	// in between we connect up hconst of check nodes and hconst+1 variable nodes

	v := state.vn.new[0]
	state.vn.Pop()

	state.H.Set(c1, v, 1)
	for h := 1; h <= hconst; h++ {
		c := state.cn.new[0]
		state.cn.Pop()

		state.H.Set(c, v, 1)

		v = state.vn.new[0]
		state.vn.Pop()

		state.H.Set(c, v, 1)
	}

	state.H.Set(c2, v, 1)
}

func initH(state *gceState, g int) {
	// we init by taking a random set of g/2 check nodes and variable nodes
	// to make a cycle with a girth equal to g.
	sort.Ints(state.cn.new)
	sort.Ints(state.vn.new)

	max := g / 2
	for i := 0; i < max; i++ {
		state.H.Set(i, i, 1)
		state.H.Set((i+1)%max, i, 1)
		state.cn.Pop()
		state.vn.Pop()
	}
}

type Nodes struct {
	old []int
	new []int
}

func newNodes(count int) *Nodes {
	nodes := Nodes{
		old: make([]int, 0),
		new: make([]int, count),
	}
	for i := 0; i < count; i++ {
		nodes.new[i] = i
	}

	return &nodes
}

func (n *Nodes) Copy() *Nodes {
	nn := &Nodes{
		old: make([]int, len(n.old)),
		new: make([]int, len(n.new)),
	}

	copy(nn.old, n.old)
	copy(nn.new, n.new)
	return nn
}

func (n *Nodes) Pop() {
	n.old = append(n.old, n.new[0])

	copy(n.new[:], n.new[1:])
	n.new = n.new[:len(n.new)-1]
}

func (n *Nodes) Exhausted() bool {
	return len(n.new) == 0
}

func (n *Nodes) Len() int {
	return len(n.new)
}

func (n *Nodes) PopAll() {
	n.old = append(n.old, n.new...)

	n.new = make([]int, 0)
}

func findTwoNodes(H mat.SparseMat, checkIndices []int, dist int) (checkNode, otherNode int, success bool) {
	//we want to start with nodes have the smallest weight
	sort.Slice(checkIndices, func(i, j int) bool {
		a := H.Row(i).HammingWeight()
		b := H.Row(j).HammingWeight()
		return a < b
	})

	for _, c1 := range checkIndices {
		x, good := getNodeAtDistFromCheck(c1, dist, H)
		if !good {
			continue
		}
		return c1, x, true
	}

	return -1, -1, false
}

func getNodeAtDistFromCheck(checkIndex, atLeastDist int, H mat.SparseMat) (nodeIndex int, success bool) {
	checkNodeHistory := make(map[int]bool)
	variableNodeHistory := make(map[int]bool)
	currentLevel := make(map[int]bool)

	// first we save the current node in history and set it to the curren level
	checkNodeHistory[checkIndex] = true
	currentLevel[checkIndex] = false // false if it's a check, true for variable node

	level := 0
	var nextLevel map[int]bool
	_, cols := H.Dims()
	for level <= 2*cols && level < atLeastDist {
		nextLevel = make(map[int]bool)

		if level%2 == 0 {
			for c := range currentLevel {
				for _, v := range H.Row(c).NonzeroArray() {
					if _, has := variableNodeHistory[v]; has {
						continue
					}
					variableNodeHistory[v] = true
					nextLevel[v] = true
				}
			}
		} else {
			for v := range currentLevel {
				for _, c := range H.Column(v).NonzeroArray() {
					if _, has := checkNodeHistory[c]; has {
						continue
					}
					checkNodeHistory[c] = true
					nextLevel[c] = false
				}
			}
		}

		level++
		currentLevel = nextLevel
	}
	if level >= atLeastDist && len(currentLevel) > 0 {
		//we're going to return one of the indices in the currentLevel
		// we will return the one with the fewest connections (randomly)

		min := -1
		for n, variable := range currentLevel {
			if min == -1 {
				min = n
				continue
			}

			var a, b int
			if variable {
				a = H.Column(n).HammingWeight()
				b = H.Column(min).HammingWeight()

			} else {
				a = H.Row(n).HammingWeight()
				b = H.Row(min).HammingWeight()
			}

			if a < b {
				min = n
			}
		}
		return min, true
	}
	return -1, false
}
