package internal

import (
	"context"
	"os"

	"github.com/cheggaaa/pb/v3"
	mat "github.com/nathanhack/sparsemat"
	"github.com/nathanhack/threadpool"
	"github.com/sirupsen/logrus"
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
	logrus.Debugf("Preparing matrix for Gaussian-Jordan Elimination")
	rows, cols := H.Dims()
	result := mat.CSRMatCopy(H)
	columnSwapHistory := make([]int, cols)

	//initialize the columnIndices
	for c := 0; c < cols; c++ {
		columnSwapHistory[c] = c
	}

	//to fail fast we first do the lower triangle then do the upper
	rank := lowerTriangular(ctx, rows, result, columnSwapHistory, threads, logrus.GetLevel() == logrus.DebugLevel)
	if rank != rows {
		logrus.Warnf("Only %v rows of %v linearly independent (diff:%v)", rank, rows, rank-rows)
	}

	// at this point we know we had all linearly independent rows!
	// and the lower triangle is done so we'll take care of the top
	upperTriangular(ctx, rows, result, threads, logrus.GetLevel() == logrus.DebugLevel)

	logrus.Debugf("Gaussian-Jordan Elimination complete")
	if rank != rows {
		result = result.Slice(0, 0, rank, cols)
	}
	return result, columnSwapHistory
}

func upperTriangular(ctx context.Context, rows int, H mat.SparseMat, threads int, showProgressBar bool) bool {
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
		if H.At(r, r) > 0 {
			eliminateOtherRowsParallel(ctx, r, H, threads)
		}
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

func eliminateOtherRowsParallel(ctx context.Context, rowIndex int, result mat.SparseMat, threads int) {
	pivots := result.Column(rowIndex).NonzeroArray()
	pool := threadpool.New(ctx, threads)

	//for all pivots != rowIndex subtract it (in GF2 subtract is add)
	// we use AddRows without a mutex, this will work only for CSRMat
	for _, index := range pivots {
		pIndex := index
		if index != rowIndex {
			pool.Add(func() {
				func(p int) {
					result.AddRows(rowIndex, p, p)
				}(pIndex)
			})
		}
	}
	pool.Wait()
}

func eliminateLowerRowsParallel(ctx context.Context, rowIndex int, result mat.SparseMat, threads int) {
	pivots := result.Column(rowIndex).NonzeroArray()
	pool := threadpool.New(ctx, threads)

	//for all pivots > rowIndex subtract it (in GF2 subtract is add)
	// we use AddRows without a mutex, this will work only for CSRMat
	for _, index := range pivots {
		pIndex := index
		if pIndex > rowIndex {
			pool.Add(func() {
				func(p int) {
					result.AddRows(rowIndex, p, p)
				}(pIndex)
			})
		}
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

	return lowerTriangular(ctx, min, tmp, columnSwapHistory, threads, showProgressBar)
}

func lowerTriangular(ctx context.Context, rows int, H mat.SparseMat, columnSwapHistory []int, threads int, showProgressBar bool) int {
	bar := pb.Full.New(rows)
	logrus.Debugf("Row echelon")
	bar.Set("prefix", "Processing Row ")
	bar.SetWriter(os.Stdout)
	if showProgressBar {
		bar.Start()
	}
	rowsWithPivots := 0
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
			continue
		}
		rowsWithPivots++

		//when here we know pivots have values and the last one
		//should be a row to switch with
		pivot := pivots[len(pivots)-1]
		H.SwapRows(r, pivot)

		// now the rth row has a pivot in the rth row and rth column
		// so we now subtract it from all other rows with a 1
		// in the rth column
		eliminateLowerRowsParallel(ctx, r, H, threads)
	}

	if rowsWithPivots != rows {
		logrus.Debugf("Consolidating %v linear dependent rows to bottom of matrix", rows-rowsWithPivots)
		// if true we reorder so that diagonal has all the ones grouped together
		// pushed to the top(first rows). As a consequence of this it means
		// linearly dependant rows are at the bottom (last rows).
		currRow := 0
	main:
		for currRow > rows {
			if H.At(currRow, currRow) == 0 {
				// find the next that  isn't zero to replace
				replaceRow := currRow + 1
				for replaceRow < rows {
					if H.At(replaceRow, replaceRow) > 0 {
						break
					}
				}
				if replaceRow == rows {
					// everything was zeros nothing to do matrix is ordered
					break main
				}

				// ok we swap these rows then also swap the columns
				// the row swap doesn't have to be tracked but the columns do
				H.SwapRows(currRow, replaceRow)
				H.SwapColumns(currRow, replaceRow) //this functions for CSR has a problem
				swapColOrder(currRow, replaceRow, columnSwapHistory)
			}
		}
	}

	bar.SetTemplateString(`{{string . "prefix"}}{{counters . }}{{string . "suffix"}}`)
	bar.Set("suffix", " Done")
	bar.Finish()

	return rowsWithPivots
}
