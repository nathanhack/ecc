package iterative

import (
	"github.com/nathanhack/ecc/linearblock/messagepassing/bec"
	mat "github.com/nathanhack/sparsemat"
)

type Simple struct {
	H           mat.SparseMat
	checkToVars [][]int
	varToChecks [][]int
}

func (s *Simple) Flip(currentCodeword []bec.ErasureBit) (nextCodeword []bec.ErasureBit, done bool) {
	if s.H == nil {
		panic("Simple BEC flipping algorithm must have the H parity matrix set before using")
	}
	if s.checkToVars == nil {
		s.init()
	}

	nextCodeword = make([]bec.ErasureBit, len(currentCodeword))
	copy(nextCodeword, currentCodeword)
	erasedBits := getErasedIndices(nextCodeword)
	progress := false

	for len(erasedBits) > 0 {

		progress = false

		checksCompleted := make(map[int]bool)
		for _, erasedBit := range erasedBits {
			for _, row := range s.varToChecks[erasedBit] {
				if _, has := checksCompleted[row]; has {
					continue
				}

				if progressM(nextCodeword, s.checkToVars[row]) {
					progress = true
				}
			}
		}

		if !progress {
			return nextCodeword, true
		}

		erasedBits = getErasedIndices(nextCodeword)
	}

	return nextCodeword, !progress
}

func (s *Simple) init() {
	if s.checkToVars != nil {
		return
	}
	rows, cols := s.H.Dims()

	s.varToChecks = make([][]int, cols)
	for v := range s.varToChecks {
		s.varToChecks[v] = make([]int, 0)
	}

	s.checkToVars = make([][]int, rows)
	for c := range s.checkToVars {
		s.checkToVars[c] = s.H.Row(c).NonzeroArray()
		for _, v := range s.checkToVars[c] {
			s.varToChecks[v] = append(s.varToChecks[v], c)
		}
	}
}

func getErasedIndices(m []bec.ErasureBit) []int {
	erasedBits := make([]int, 0, len(m))

	for i, r := range m {
		if r == bec.Erased {
			erasedBits = append(erasedBits, i)
		}
	}
	return erasedBits
}

func progressM(M []bec.ErasureBit, B []int) bool {
	count := 0
	missing := -1
	value := 0
	for _, b := range B {
		if M[b] != bec.Erased {
			value += int(M[b])
			continue
		}

		count++
		missing = b
		//we can only fix a check node with only 1 missing value
		if count > 1 {
			return false
		}
	}

	if count != 1 {
		return false
	}
	M[missing] = bec.ErasureBit(value % 2)

	return true
}
