package iterative

import (
	"github.com/nathanhack/errorcorrectingcodes/linearblock/messagepassing/bec"
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

////BEC using the linear block it will try and recover any erased (Erased) bits.  All received values of
//// One(1) or Zero(0) will be assumed correct.  Every element in received is expected to contain one
//// of the following: One(1), Zero(0), or Erased(2). Any other values will have undetermined csv.
//func BEC(lb *linearblock.LinearBlock, received []bec.ErasureBit) []bec.ErasureBit {
//
//	//M will be used to hold the message,and will be ultimately returned
//	M := make([]bec.ErasureBit, len(received))
//	for i, r := range received {
//		M[i] = r
//	}
//
//	rows, _ := lb.H.Dims()
//	cache := make([][]int, rows)
//	for i := range cache {
//		cache[i] = lb.H.Row(i).NonzeroArray()
//	}
//
//	progress := true
//	for progress && hasErasedBits(M) {
//		progress = false
//		for i := 0; i < rows; i++ {
//			B := cache[i]
//			if progressM(M, B) {
//				progress = true
//			}
//		}
//	}
//
//	//we make no claim on what is in M and return as is
//	// hopefully some or all of the erased bit were recovered
//	return M
//}

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
