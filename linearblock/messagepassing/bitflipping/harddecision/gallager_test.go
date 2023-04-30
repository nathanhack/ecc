package harddecision

import (
	"context"
	"github.com/nathanhack/ecc/linearblock/hamming"
	mat "github.com/nathanhack/sparsemat"
	"strconv"
	"testing"
)

func TestGallager_BitFlippingHammingCodes(t *testing.T) {
	block, err := hamming.New(context.Background(), 3, 0)
	if err != nil {
		t.Fatalf("expected no error but found: %v", err)
	}
	tests := []struct {
		message          mat.SparseVector
		flipCodewordBits []int
		maxIter          int
	}{
		{mat.DOKVec(4, 1, 0, 1, 1), []int{0}, 20},
		{mat.DOKVec(4, 1, 0, 1, 1), []int{1}, 20},
		{mat.DOKVec(4, 1, 0, 1, 1), []int{2}, 20},
		{mat.DOKVec(4, 1, 0, 1, 1), []int{3}, 20},
		{mat.DOKVec(4, 1, 0, 1, 1), []int{4}, 20},
		{mat.DOKVec(4, 1, 0, 1, 1), []int{5}, 20},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			alg := &Gallager{
				H: block.H,
			}

			//create codeword
			codeword := block.Encode(test.message)
			expected := mat.CSRVecCopy(codeword)

			for _, index := range test.flipCodewordBits {
				codeword.Set(index, codeword.At(index)+1)
			}

			actual := BitFlipping(alg, block.H, codeword, test.maxIter)

			if !actual.Equals(expected) {
				t.Fatalf("expected %v but found %v", expected, actual)
			}
		})
	}
}

func BenchmarkGallager_BitFlipping(b *testing.B) {
	h := mat.CSRMat(4, 6, 1, 1, 0, 1, 0, 0, 0, 1, 1, 0, 1, 0, 1, 0, 0, 0, 1, 1, 0, 0, 1, 1, 0, 1)
	g := &Gallager{
		H: h,
	}
	input := mat.CSRVec(6, 1, 0, 1, 0, 1, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BitFlipping(g, h, input, 1)
	}
}
