package internal

import (
	mat "github.com/nathanhack/sparsemat"
)

func ColumnSwapped(H mat.SparseMat, order []int) mat.SparseMat {
	rows, cols := H.Dims()
	result := mat.CSRMat(rows, cols)

	for c, c1 := range order {
		result.SetColumn(c, H.Column(c1))
	}
	return result
}

// ValidateHGMatrices tests if G*H.T ==0 where H.T is the transpose of H
func ValidateHGMatrices(G, H mat.SparseMat) bool {
	rows, _ := G.Dims()
	cols, _ := H.Dims()

	//we cache the H.T hopefully this is in CSR so this should be way
	// faster than taking the actual H.T() then doing this
	cache := make([]mat.SparseVector, cols)
	for i := 0; i < cols; i++ {
		cache[i] = H.Row(i)
	}
	for i := 0; i < rows; i++ {
		row := G.Row(i)
		for j := 0; j < cols; j++ {
			//equiv to G*H.T
			if row.Dot(cache[j]) > 0 {
				return false
			}
		}
	}

	return true
}
