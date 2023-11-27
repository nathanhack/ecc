package gce

import (
	"context"
	"testing"

	"github.com/nathanhack/ecc/linearblock"
	"github.com/nathanhack/ecc/linearblock/internal"
	mat "github.com/nathanhack/sparsemat"
	"github.com/sirupsen/logrus"
)

func TestGeneral(t *testing.T) {
	girth := 24
	checked := false
	checkpoint := func(currentBest *linearblock.LinearBlock) {
		checked = true
	}

	l, err := Search(context.Background(), 1020, 2040, girth, 1, 0, true, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	actual := linearblock.CalculateGirth(context.Background(), l.H, 0)
	if girth != actual {
		t.Fatalf("expected %v but found %v", girth, actual)
	}

	if !internal.ValidateHGMatrices(l.Processing.G, internal.ColumnSwapped(l.H, l.Processing.HColumnOrder)) {
		t.Fatalf("expected linearblock to validate")
	}

	if !checked {
		t.Fatalf("expected checkpoint to be true")
	}

	t.Logf("Final H check weight counts: %v", matrixCheckWeightCounts(l.H))
	logrus.Infof("Final H var weight counts: %v", matrixVarWeightCounts(l.H))
	t.Logf("Final H var weight counts: %v", matrixVarWeightCounts(l.H))
}

func BenchmarkSearch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Search(context.Background(), 1002, 2004, 22, 1, 0, true, nil)
		if err != nil {
			b.Fatalf("expected no errors:%v", err)
		}
	}
}

func matrixCheckWeightCounts(H mat.SparseMat) map[int]int {
	weightCounts := make(map[int]int)
	rows, _ := H.Dims()
	for r := 0; r < rows; r++ {
		w := H.Row(r).HammingWeight()
		weightCounts[w]++
	}
	return weightCounts
}

func matrixVarWeightCounts(H mat.SparseMat) map[int]int {
	weightCounts := make(map[int]int)
	_, columns := H.Dims()
	for c := 0; c < columns; c++ {
		w := H.Column(c).HammingWeight()
		weightCounts[w]++
	}
	return weightCounts
}
