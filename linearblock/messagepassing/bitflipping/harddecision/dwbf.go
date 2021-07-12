package harddecision

import (
	"fmt"
	mat "github.com/nathanhack/sparsemat"
	mat2 "gonum.org/v1/gonum/mat"
)

//DWBF_F is a single bit flipping hard decision alg based on the paper
//"Dynamic Weighted Bit-Flipping Decoding Algorithms for LDPC Codes"
// by Tofar C.-Y. Chang and Yu T. Su
type DWBF_F struct {
	AlphaFactor  float64 //α:  0 < α < 1
	EtaThreshold float64 //η: no requirement but frequently 0.0 is a good value
	H            mat.SparseMat
	z            mat.SparseVector //original codeword
	r            *mat2.Dense
	e_n          []float64
	columnCache  [][]int
	rowCache     [][]int
}

func argMaxFloat(values []float64) int {
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

func (D *DWBF_F) init(codeword mat.SparseVector) {
	if D.AlphaFactor <= 0 || 1 <= D.AlphaFactor {
		panic(fmt.Sprintf("0<α<1 is required but found %v ", D.AlphaFactor))
	}
	D.z = mat.CSRVecCopy(codeword)

	rows, cols := D.H.Dims()
	D.r = mat2.NewDense(rows, cols, nil)

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			D.r.Set(i, j, 1)
		}
	}
	D.e_n = make([]float64, cols)

	if D.rowCache == nil {
		D.rowCache = make([][]int, cols)
		for i := 0; i < rows; i++ {
			D.rowCache[i] = D.H.Row(i).NonzeroArray()
		}
	}

	if D.columnCache == nil {
		D.columnCache = make([][]int, cols)
		for j := 0; j < cols; j++ {
			D.columnCache[j] = D.H.Column(j).NonzeroArray()
		}
	}
}
func (D *DWBF_F) Reset() {
	D.z = nil
	D.r = nil
}
func (D *DWBF_F) Flip(currentSyndromes mat.SparseVector, currentCodeword mat.SparseVector) (nextCodeword mat.SparseVector, done bool) {
	//first we check if the syndromes was zero
	if currentSyndromes.IsZero() {
		return currentCodeword, true
	}

	//internally we keep state
	// and if the first time we init()
	if D.z == nil || D.r == nil {
		D.init(currentCodeword)
	}

	// if there are errors then we calculate the
	// flipping function E_n vector and r matrix
	D.nextE_n(currentSyndromes, currentCodeword)

	// with the updated E_n we now determine the bit set
	// B = {n|n arg max_i E_i}
	n := argMaxFloat(D.e_n)

	// then let E_n = -E_n
	for i, e := range D.e_n {
		D.e_n[i] = -e
	}

	// and we flip that bit
	nextCodeword = mat.CSRVecCopy(currentCodeword)
	nextCodeword.Set(n, nextCodeword.At(n)+1)

	D.nextR()
	return nextCodeword, false
}

func (D *DWBF_F) nextE_n(syndromes mat.SparseVector, codeword mat.SparseVector) {
	// where l is the iteration
	// E^(l)_n = -(1-2*z_n)*(1-2*u_n)-α * sum(r^(l-1)_{mn}*(1-2*s_m), m ∈ M(n))

	synds := syndromes.NonzeroArray()
	syndsLen := len(synds)
	for n := 0; n < D.z.Len(); n++ {
		sum := 0.0
		cacheRow := D.columnCache[n]
		cacheRowLen := len(cacheRow)
		for i, j := 0, 0; i < cacheRowLen && j < syndsLen; {
			if cacheRow[i] == synds[j] {
				sum += -D.r.At(cacheRow[i], n)
				i++
				j++
			} else if cacheRow[i] < synds[j] {
				sum += D.r.At(cacheRow[i], n)
				i++
			} else {
				j++
			}
		}
		D.e_n[n] = -float64((1-2*D.z.At(n))*(1-2*codeword.At(n))) - D.AlphaFactor*sum
	}
}

func (D *DWBF_F) nextR() {
	// r^(l)_{mn} = min( thresh(-E^(l)_n'), n' ∈ N(m)\n)
	rows, cols := D.r.Dims()
	for m := 0; m < rows; m++ {
		for n := 0; n < cols; n++ {
			min := 0.0
			minIndex := -1
			for _, n1 := range D.rowCache[m] {
				if n == n1 {
					continue
				}
				v := threshold(-D.e_n[n1], D.EtaThreshold)
				if minIndex == -1 || min > v {
					min = v
					minIndex = n1
				}
			}
			D.r.Set(m, n, min)
		}
	}
}

func threshold(value, thresh float64) float64 {
	if value >= thresh {
		return value - thresh
	}
	return 0
}
