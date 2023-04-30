package iterative

import (
	"github.com/nathanhack/ecc/linearblock/messagepassing/bec"
	mat "github.com/nathanhack/sparsemat"
)

type Simple struct {
	H     mat.SparseMat
	cache [][]int
}

func (s *Simple) Flip(currentCodeword []bec.ErasureBit) (nextCodeword []bec.ErasureBit, done bool) {
	if s.H == nil {
		panic("Simple BEC flipping algorithm must have the H parity matrix set before using")
	}
	if s.cache == nil {
		s.init()
	}

	nextCodeword = copyErasureBits(currentCodeword)

	if !hasErasedBits(nextCodeword) {
		return nextCodeword, true
	}

	progress := false
	for _, B := range s.cache {
		if progressM(nextCodeword, B) {
			progress = true
		}
	}

	done = !(progress && hasErasedBits(currentCodeword))
	return nextCodeword, done
}

func (s *Simple) init() {
	if s.cache != nil {
		return
	}
	rows, _ := s.H.Dims()
	s.cache = make([][]int, rows)
	for i := range s.cache {
		s.cache[i] = s.H.Row(i).NonzeroArray()
	}
}

func copyErasureBits(m []bec.ErasureBit) []bec.ErasureBit {
	M := make([]bec.ErasureBit, len(m))
	for i, r := range m {
		M[i] = r
	}
	return M
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

func hasErasedBits(M []bec.ErasureBit) bool {
	for i := 0; i < len(M); i++ {
		if M[i] == bec.Erased {
			return true
		}
	}
	return false
}
