package internal

import (
	"context"
	"github.com/cheggaaa/pb/v3"
	mat "github.com/nathanhack/sparsemat"
	"github.com/nathanhack/threadpool"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
)

func swapColOrder(i, j int, colIndices []int) {
	x := len(colIndices)
	if 0 <= i && i < x && 0 <= j && j < x {
		idx := colIndices[i]
		colIndices[i] = colIndices[j]
		colIndices[j] = idx
	}
}

func findPivotColGF2(H mat.SparseMat, forRow int) int {
	rows, _ := H.Dims()

	for r := forRow; r < rows; r++ {
		row := H.Row(r).NonzeroArray()
		if len(row) == 0 {
			continue
		}

		col := row[len(row)-1]
		if col > forRow {
			return col
		}
	}
	return -1
}

func GaussianJordanEliminationGF2(ctx context.Context, H mat.SparseMat, threads int) (mat.SparseMat, []int) {
	rows, cols := H.Dims()
	result := mat.CSRMatCopy(H)
	columnSwapHistory := make([]int, cols)

	//initialize the columnIndices
	for c := 0; c < cols; c++ {
		columnSwapHistory[c] = c
	}

	if cols < rows {
		//null space must equal the rank
		return nil, nil
	}

	//to fail fast we first do the lower triangle then do the upper
	if lowerTrianglar(ctx, rows, result, columnSwapHistory, threads, logrus.GetLevel() == logrus.DebugLevel) != rows {
		logrus.Debugf("All rows not linearly independant")
		return nil, nil
	}

	// at this point we know we had all linearly independent rows!
	// and the lower triangle is done so we'll take care of the top
	if !upperTrianglar(ctx, rows, result, threads, logrus.GetLevel() == logrus.DebugLevel) {
		return nil, nil
	}

	logrus.Debugf("Gaussian-Jordan Elimination complete")
	return result, columnSwapHistory
}

func upperTrianglar(ctx context.Context, rows int, H mat.SparseMat, threads int, showProgressBar bool) bool {
	bar := pb.Full.New(rows)
	logrus.Debugf("Reduced row echelon")
	bar.Set("prefix", "Processing Row ")
	bar.SetWriter(os.Stdout)
	if showProgressBar {
		bar.Start()
	}

	for r := 0; r < rows; r++ {
		bar.Increment()
		select {
		case <-ctx.Done():
			return false
		default:
		}
		eliminateOtherRows(ctx, r, H, threads)
	}
	bar.SetTemplateString(`{{string . "prefix"}}{{counters . }}{{string . "suffix"}}`)
	bar.Set("suffix", " Done")
	bar.Finish()
	return true
}

func pivotsSwapReturn(H mat.SparseMat, rowIndex int, columnsSwapHistory []int) []int {
	pivots := H.Column(rowIndex).NonzeroArray()
	if len(pivots) == 0 || pivots[len(pivots)-1] < rowIndex {
		// there aren't any where we need them
		// so we'll do a columns swap to get what we need
		colPivot := findPivotColGF2(H, rowIndex)
		if colPivot == -1 {
			//we get here when there aren't any more non zero rows
			//so this matrix null space doesn't span the rank
			return nil
		}

		H.SwapColumns(rowIndex, colPivot) //this functions for CSR has a problem
		swapColOrder(rowIndex, colPivot, columnsSwapHistory)

		//now redo pivots
		pivots = H.Column(rowIndex).NonzeroArray()
	}
	return pivots
}

func eliminateOtherRows(ctx context.Context, rowIndex int, result mat.SparseMat, threads int) {
	//create a pool with 1 less pivot
	pivots := result.Column(rowIndex).NonzeroArray()
	pool := threadpool.New(ctx, threads, len(pivots)-1)
	rrow := result.Row(rowIndex)
	mut := sync.RWMutex{}

	//for all pivots except the one equal to r subtract it (in GF2 subtract is add)
	for _, index := range pivots {
		pIndex := index
		if index != rowIndex {
			pool.Add(func() {
				func(p int) {
					mut.RLock()
					prow := result.Row(p)
					mut.RUnlock()
					prow.Add(prow, rrow)
					mut.Lock()
					result.SetRow(p, prow)
					mut.Unlock()
				}(pIndex)
			})
		}
	}
	pool.Wait()
}

func eliminateLowerRows(ctx context.Context, rowIndex int, result mat.SparseMat, threads int) {
	//create a pool with 1 less pivot
	pivots := result.Column(rowIndex).NonzeroArray()
	pool := threadpool.New(ctx, threads, len(pivots))
	rrow := result.Row(rowIndex)
	mut := sync.RWMutex{}

	//for all pivots except the one equal to r subtract it (in GF2 subtract is add)
	for _, index := range pivots {
		pIndex := index
		pool.Add(func() {
			if pIndex > rowIndex {
				func(p int) {
					mut.RLock()
					prow := result.Row(p)
					mut.RUnlock()
					prow.Add(prow, rrow)
					mut.Lock()
					result.SetRow(p, prow)
					mut.Unlock()
				}(pIndex)
			}
		})
	}
	pool.Wait()
}

func CalculateRank(ctx context.Context, H mat.SparseMat, threads int, showProgressBar bool) int {
	if H == nil {
		return -1
	}

	tmp := mat.CSRMatCopy(H)

	rows, cols := H.Dims()

	min := rows
	if cols < rows {
		min = cols
	}
	columnSwapHistory := make([]int, cols)

	return lowerTrianglar(ctx, min, tmp, columnSwapHistory, threads, showProgressBar)
}

func lowerTrianglar(ctx context.Context, rows int, H mat.SparseMat, columnSwapHistory []int, threads int, showProgressBar bool) int {
	bar := pb.Full.New(rows)
	logrus.Debugf("Row echelon")
	bar.Set("prefix", "Processing Row ")
	bar.SetWriter(os.Stdout)
	if showProgressBar {
		bar.Start()
	}

	for r := 0; r < rows; r++ {
		select {
		case <-ctx.Done():
			return -1
		default:
		}
		bar.Increment()
		//we process both upper and lower triangle at the same time
		//first find pivots for the column equal to r
		pivots := pivotsSwapReturn(H, r, columnSwapHistory)
		if pivots == nil {
			return r
		}

		//when here we know pivots have values and the last one
		//should be a row to switch with
		pivot := pivots[len(pivots)-1]
		H.SwapRows(r, pivot)

		// now the rth row has a pivot in the rth row and rth column
		// so we now subtract it from all other rows with a 1
		// in the rth column
		eliminateLowerRows(ctx, r, H, threads)
	}

	bar.SetTemplateString(`{{string . "prefix"}}{{counters . }}{{string . "suffix"}}`)
	bar.Set("suffix", " Done")
	bar.Finish()

	return rows
}
