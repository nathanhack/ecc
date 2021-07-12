package gallager

import (
	"context"
	"fmt"
	"github.com/nathanhack/errorcorrectingcodes/linearblock"
	"github.com/nathanhack/errorcorrectingcodes/linearblock/internal"
	mat "github.com/nathanhack/sparsemat"
	"github.com/sirupsen/logrus"
	"math/rand"
)

//GallagerRateInput takes in the message input size in bits (m), the column weight (wc), and row weight (wr)
func Search(ctx context.Context, m, wc, wr, smallestCycleAllowed, maxIter, threads int) (lb *linearblock.LinearBlock, err error) {
	if 3 > wc {
		return nil, fmt.Errorf("wc must be greater than or equal to 3")
	}
	if wc >= wr {
		return nil, fmt.Errorf("wc (%v) must be less than wr (%v)", wc, wr)
	}
	if m%wc != 0 {
		return nil, fmt.Errorf("wc (%v) must divide m (%v)", wc, m)
	}
	if smallestCycleAllowed%2 != 0 {
		return nil, fmt.Errorf("smallestCycle must be an even number")
	}
	if smallestCycleAllowed < 4 {
		return nil, fmt.Errorf("smallestCycle must at least 4")
	}

	N := m / wc * wr
	K := m / wc
	// Prepare a HPrime used to create all H'subs
	HPrime := mat.DOKMat(K, N)
	for i := 0; i < K; i++ {
		offset := i * wr
		for col := 0; col < wr; col++ {
			HPrime.Set(i, col+offset, 1)
		}
	}
	iter := maxIter

	for iter > 0 {
		iter, lb, err = search(ctx, N, K, m, wc, iter, smallestCycleAllowed, threads, HPrime)
	}
	return
}

func search(ctx context.Context, N, K, m, wc, iter, smallestCycleAllowed, threads int, HPrime mat.SparseMat) (int, *linearblock.LinearBlock, error) {
	//make the real parity matrix, we'll fill it with the
	// correct data next
	H := mat.DOKMat(m, N)

	// H is made of three subs
	// the first sub == HPrime
	// the others are permuted(HPrime)
	// we're guaranteed that we can remove
	// 4 cycles so if not allow we just
	// loop again eventually it'll provide
	// permutation that works

	s := 0
	for s < wc && iter > 0 {
		iter--
		logrus.Debugf("Iterations remaining %v", iter)
		sub := HPrime
		if s > 0 {
			sub = permuteColumns(HPrime)
		}
		setSubH(H, sub, s)

		calGirth := linearblock.CalculateGirthLowerBound(H, smallestCycleAllowed, threads)
		if -1 < calGirth && calGirth < smallestCycleAllowed {
			continue
		}

		rank := internal.CalculateRank(ctx, H, threads, false)
		if rank != (s+1)*K {
			continue
		}
		s++
	}
	if s != wc {
		return iter, nil, fmt.Errorf("failed to find a solution")
	}
	logrus.Debugf("Gallager H Matrix found")

	order, g := internal.NewFromH(ctx, H, threads)
	if order == nil {
		return iter, nil, fmt.Errorf("unable to create generator for H matrix")
	}

	return 0, &linearblock.LinearBlock{
		H: H,
		Processing: &linearblock.Systemic{
			HColumnOrder: order,
			G:            g,
		},
	}, nil
}

func permuteColumns(H mat.SparseMat) mat.SparseMat {
	rows, cols := H.Dims()
	result := mat.DOKMat(rows, cols)

	//make indices to do permutation
	idx := make([]int, cols)
	for i := 0; i < cols; i++ {
		idx[i] = i
	}

	//shuffle them
	rand.Shuffle(len(idx), func(i, j int) {
		tmp := idx[i]
		idx[i] = idx[j]
		idx[j] = tmp
	})

	//now set the columns
	for i, col := range idx {
		tmp := H.Column(col)
		result.SetColumn(i, tmp)
	}

	return result
}

func setSubH(H, Hsub mat.SparseMat, index int) {
	K, _ := Hsub.Dims()
	offset := index * K
	H.SetMatrix(Hsub, offset, 0)
}
