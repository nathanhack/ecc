package harddecision

import (
	mat "github.com/nathanhack/sparsemat"
)

func argMaxInt(values []int) int {
	result := 0
	max := values[0]
	for i := 1; i < len(values); i++ {
		v := values[i]
		if max < v {
			result = i
			max = v
		}
	}
	return result
}

type Gallager struct {
	H        mat.SparseMat
	e_n      []int
	rowCache [][]int
}

func (g *Gallager) Reset() {
	//nothing to do here
}

func (g *Gallager) Flip(currentSyndromes mat.SparseVector, currentCodeword mat.SparseVector) (nextCodeword mat.SparseVector, done bool) {
	if g.H == nil {
		panic("Gallager H matrix must be set before calling Algorithm")
	}
	//first we check if the syndromes was zero
	if currentSyndromes.IsZero() {
		return currentCodeword, true
	}

	if g.e_n == nil {
		g.init(currentCodeword.Len())
	}

	//since there was errors(syndrome is not zero) we'll
	// calculate the flipping function E_n vector
	g.nextE_n(currentSyndromes)

	n := argMaxInt(g.e_n)

	// and we flip that bit
	nextCodeword = mat.CSRVecCopy(currentCodeword)
	nextCodeword.Set(n, nextCodeword.At(n)+1)

	return nextCodeword, false
}

func (g *Gallager) nextE_n(syndromes mat.SparseVector) {
	// E_n = -sum((1-2*s_m), m ∈ M(n))

	synIndices := syndromes.NonzeroArray()
	synIndicesLen := len(synIndices)
	for n := 0; n < len(g.e_n); n++ {
		sum := 0
		indices := g.rowCache[n]
		indicesLen := len(indices)
		for i, j := 0, 0; i < indicesLen && j < synIndicesLen; {
			if indices[i] == synIndices[j] {
				sum++
				i++
				j++
			} else if indices[i] < synIndices[j] {
				i++
			} else {
				j++
			}
		}

		g.e_n[n] = -len(indices) + 2*sum
	}
}

func (g *Gallager) init(codewordLen int) {
	if g.e_n != nil {
		return
	}

	g.e_n = make([]int, codewordLen)
	g.rowCache = make([][]int, codewordLen)
	for n := 0; n < len(g.e_n); n++ {
		g.rowCache[n] = g.H.Column(n).NonzeroArray()
	}
}
