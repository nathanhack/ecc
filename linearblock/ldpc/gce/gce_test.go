package gce

import (
	"context"
	"testing"

	"github.com/nathanhack/ecc/linearblock"
	"github.com/nathanhack/ecc/linearblock/internal"
)

func TestGeneral(t *testing.T) {
	girth := 22
	checked := false
	checkpoint := func(currentBest *linearblock.LinearBlock) {
		checked = true
	}

	l, err := Search(context.Background(), 102, 204, girth, 1, 0, true, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	actual := linearblock.CalculateGirth(l.H, 0)
	if girth != actual {
		t.Fatalf("expected %v but found %v", girth, actual)
	}

	if !internal.ValidateHGMatrices(l.Processing.G, internal.ColumnSwapped(l.H, l.Processing.HColumnOrder)) {
		t.Fatalf("expected linearblock to validate")
	}

	if !checked {
		t.Fatalf("expected checkpoint to be true")
	}
}

func BenchmarkSearch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Search(context.Background(), 1002, 2004, 22, 1, 0, true, nil)
		if err != nil {
			b.Fatalf("expected no errors:%v", err)
		}
	}
}
