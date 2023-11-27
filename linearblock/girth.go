package linearblock

import (
	"context"
	"math"
	"sync"

	mat "github.com/nathanhack/sparsemat"
	"github.com/nathanhack/threadpool"
)

// CalculateGirthLowerBoundByEdges returns a bool if true it is possible for the matrix to be
// free of all cycles C_i  where i \in 3<=i<=minGirth. If false, it is impossible to
// be free of all cycles C_i.
func CalculateGirthLowerBoundByEdges(m mat.SparseMat, minGirth int) bool {
	// from the paper Fast Distributed Algorithms for Girth, Cycles and Small Subgraphs by K. Censor-Hillel, et al
	// that if m is free of all C_i cycles 3<=i<=2k then m contains at most n^(1+1/k)+n edges.

	rows, cols := m.Dims()
	edges := 0
	for i := 0; i < rows; i++ {
		edges += len(m.Row(i).NonzeroMap())
	}

	n := float64(rows + cols)
	n = math.Pow(n, 1+1/(float64(minGirth)/2)) + n
	if int(n) < edges {
		return false
	}
	return true
}

type girthNode struct {
	parentIndex int
}

// CalculateGirth calculates the girth of the tanner csv induced by m.
// threads specifies the number of threads to use if <=0 will use runtime.NumCPU()
func CalculateGirth(ctx context.Context, m mat.SparseMat, threads int) int {
	return CalculateGirthLowerBound(ctx, m, -1, threads)
}

// CalculateGirthLowerBound returns the length of the smallest cycle.
// It searches for cycles with a length <= smallestGirth. If no cycles are found
// that are smaller or equal to smallestGirth then it returns -1.
// threads specifies the number of threads to use if <=0 will use runtime.NumCPU()
func CalculateGirthLowerBound(ctx context.Context, m mat.SparseMat, smallestGirth, threads int) int {
	if smallestGirth != -1 && (smallestGirth < 4 || smallestGirth%2 != 0) {
		panic("smallestGirth == -1 or smallestGirth must be a even number >=4")
	}

	rows, _ := m.Dims()

	pool := threadpool.New(ctx, threads)
	calculated := -1
	mux := sync.RWMutex{}
	for i := 0; i < rows; i++ {
		index := i //
		pool.Add(func() {
			mux.RLock()
			g := CalculateCycleLowerBound(ctx, m, index, smallestGirth)
			mux.RUnlock()

			mux.Lock()
			if g > 0 && (g <= smallestGirth || smallestGirth == -1) {
				smallestGirth = g
				calculated = g
			}
			mux.Unlock()
		})
	}
	pool.Wait()
	return calculated
}

// HasGirthSmallerThan will search for cycle smaller than the given cycleLen.
// Return true if it found a cycle smaller than cycleLen, else returns false.
// threads specifies the number of threads to use if <=0 will use runtime.NumCPU()
func HasGirthSmallerThan(ctx context.Context, m mat.SparseMat, cycleLen, threads int) bool {
	if cycleLen != -1 && cycleLen < 4 {
		panic("cycleLen == -1 or cycleLen >=4 required")
	}

	rows, _ := m.Dims()
	pool := threadpool.New(ctx, threads)
	smaller := false
	mux := sync.RWMutex{}
	for i := 0; i < rows; i++ {
		index := i
		pool.Add(func() {
			mux.RLock()
			if smaller {
				// nothing to do here
				mux.RUnlock()
				return
			}
			g := CalculateCycleLowerBound(ctx, m, index, cycleLen)

			mux.RUnlock()
			mux.Lock()
			if g > 0 && g < cycleLen {
				smaller = true
			}
			mux.Unlock()
		})
	}
	pool.Wait()
	return smaller
}

// CalculateCycleLowerBound runs a BFS starting at the checkIndex check node, for maxGirth/2 steps
// if maxGirth ==-1 it will search until it finds a cycle
// in either case it returns the length of the cycle (up to maxGirth) or -1 if no cycle was found
func CalculateCycleLowerBound(ctx context.Context, m mat.SparseMat, checkIndex, maxGirth int) int {
	if maxGirth == -1 {
		maxGirth = math.MaxInt64
	}
	//we make a history that will alternate between variable nodes and check nodes
	// as we extend to each new hop away from the checkIndex
	history := make([]map[int]girthNode, 0)
	rows, _ := m.Dims()

	//we prime the history
	hop := make(map[int]girthNode)
	for _, i := range m.Row(checkIndex).NonzeroArray() {
		hop[i] = girthNode{parentIndex: checkIndex}
	}
	//if there was only one variable node (or less than 1) then there is no way
	// this will have a loop
	if len(hop) <= 1 {
		return -1
	}
	history = append(history, hop)

	for level := 1; level < 2*rows && level < maxGirth/2+1; level++ {
		prevHop := history[level-1]
		hop := make(map[int]girthNode)
		for v, gn := range prevHop {
			levelHop := level % 2
			var indices []int
			if levelHop == 0 {
				indices = m.Row(v).NonzeroArray()
			} else {
				indices = m.Column(v).NonzeroArray()
			}
			for _, i := range indices {
				if i == gn.parentIndex {
					continue
				}
				_, has := hop[i]
				if has || (levelHop == 1 && i == checkIndex) {
					return (level + 1) * 2
				}
				hop[i] = girthNode{parentIndex: v}
			}
		}
		history = append(history, hop)
	}
	return -1
}
