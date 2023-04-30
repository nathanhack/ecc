package hamming

import (
	"context"
	"fmt"

	"github.com/nathanhack/ecc/linearblock"
	"github.com/nathanhack/ecc/linearblock/internal"
	mat "github.com/nathanhack/sparsemat"
)

// New creates the systematic hamming code with paritySymbols number of parity symbols.
// Hamming codes can detect up to two-bit errors or correct one-bit errors without
// detection of uncorrected errors.
func New(ctx context.Context, paritySymbols int, threads int) (*linearblock.LinearBlock, error) {
	if paritySymbols < 3 {
		panic("hamming codes require >=3 parity symbols")
	}
	n := 1<<paritySymbols - 1
	//k := n - paritySymbols
	H := mat.CSRMat(paritySymbols, n)

	//To make Hamming codes we make the columns the bit versions
	// of every number from 1 to and including n -> [1,n] (note they're nonzero)
	for i := 1; i <= n; i++ {
		vec := mat.CSRVec(paritySymbols)
		for j := 0; j < paritySymbols; j++ {
			if i&(1<<j) > 0 {
				vec.Set(j, 1)
			}
		}
		H.SetColumn(i-1, vec)
	}

	order, g := internal.NewFromH(ctx, H, threads)
	if order == nil {
		return nil, fmt.Errorf("unable to create generator for H matrix")
	}

	return &linearblock.LinearBlock{
		H: H,
		Processing: &linearblock.Systemic{
			HColumnOrder: order,
			G:            g,
		},
	}, nil
}
