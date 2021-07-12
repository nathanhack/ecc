package internal

import (
	"context"
	"fmt"
	mat "github.com/nathanhack/sparsemat"
	"github.com/sirupsen/logrus"
)

func ExtractAFromH(ctx context.Context, H mat.SparseMat, threads int) (A mat.SparseMat, columnOrdering []int) {
	m, N := H.Dims()

	gje, ordering := GaussianJordanEliminationGF2(ctx, H, threads)

	if gje == nil {
		return nil, nil
	}

	//let's check if we got a [ I, * ] format
	actual := gje.Slice(0, 0, m, m)
	ident := mat.CSRIdentity(m)
	if !actual.Equals(ident) {
		logrus.Errorf("failed to create transform H matrix into [I,*]")
		for i := 0; i < m; i++ {
			x := actual.Row(i).NonzeroArray()
			if len(x) > 0 {
				fmt.Println(i, ":", x)
			}
		}
		return nil, nil
	}

	//we need to convert gje from [ I, A] to [ A, I] (while keeping track)
	// and then extract A

	// first the keeping track part
	columnOrdering = make([]int, len(ordering))
	copy(columnOrdering[0:N-m], ordering[m:N])
	copy(columnOrdering[N-m:N], ordering[0:m])

	//finally extract the A
	A = gje.Slice(0, m, m, N-m)
	return
}

//NewFromH creates a derived H an G matrix containing the systematic form of parity matrix H and generator G.
// Note: the LinearBlock parity matrix's columns may be swapped.
func NewFromH(ctx context.Context, H mat.SparseMat, threads int) (HColumnOrder []int, G mat.SparseMat) {
	hrows, hcols := H.Dims()
	if hrows >= hcols {
		panic("H matrix shape == (rows, cols) where rows < cols required")
	}
	// So we now take the current H matrix
	// convert H=[*] -> H=[A,I]
	// then extract out the A and keep track of columnSwaps during it
	logrus.Debugf("Creating generator matrix from H matrix")
	A, columnSwaps := ExtractAFromH(ctx, H, threads)
	if A == nil {
		logrus.Debugf("Unable to create generator matrix from H")
		return nil, nil
	}

	AT := A.T() // transpose of A
	atRows, atCols := AT.Dims()

	//Next using A make G=[I, A^T] where A^T is the transpose of A
	G = mat.DOKMat(atRows, atRows+atCols)
	G.SetMatrix(mat.CSRIdentity(atRows), 0, 0)
	G.SetMatrix(AT, 0, atRows)

	logrus.Debugf("Generator Matrix complete")
	return columnSwaps, G
}

func ColumnSwapped(H mat.SparseMat, order []int) mat.SparseMat {
	rows, cols := H.Dims()
	result := mat.CSRMat(rows, cols)

	for c, c1 := range order {
		result.SetColumn(c, H.Column(c1))
	}
	return result
}

//ValidateHGMatrices tests if G*H.T ==0 where H.T is the transpose of H
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
