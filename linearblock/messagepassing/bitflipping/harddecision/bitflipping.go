package harddecision

import (
	mat "github.com/nathanhack/sparsemat"
)

type BitFlippingAlg interface {
	Flip(currentSyndromes mat.SparseVector, currentCodeword mat.SparseVector) (nextCodeword mat.SparseVector, done bool)
	Reset() //resets internal state for next codeword
}

func BitFlipping(bitFlippingAlg BitFlippingAlg, H mat.SparseMat, codeword mat.SparseVector, maxIter int) (result mat.SparseVector) {
	done := false
	rows, _ := H.Dims()
	result = mat.CSRVecCopy(codeword)
	syndrome := mat.CSRVec(rows)
	for i := 0; i < maxIter && !done; i++ {
		syndrome.MatMul(H, result)
		result, done = bitFlippingAlg.Flip(syndrome, result)
	}
	return result
}
