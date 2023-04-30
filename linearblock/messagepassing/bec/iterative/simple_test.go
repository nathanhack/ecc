package iterative

import (
	"github.com/nathanhack/ecc/linearblock/messagepassing/bec"
	mat "github.com/nathanhack/sparsemat"
	"reflect"
	"strconv"
	"testing"
)

func TestLinearBlock_BEC(t *testing.T) {
	tests := []struct {
		H        mat.SparseMat
		codeword []bec.ErasureBit
		expected []bec.ErasureBit
	}{
		{mat.DOKMat(4, 6, 1, 1, 0, 1, 0, 0, 0, 1, 1, 0, 1, 0, 1, 0, 0, 0, 1, 1, 0, 0, 1, 1, 0, 1), []bec.ErasureBit{0, 0, 1, bec.Erased, bec.Erased, bec.Erased}, []bec.ErasureBit{0, 0, 1, 0, 1, 1}},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			alg := &Simple{
				H: test.H,
			}

			actual := bec.Flipping(alg, test.codeword)
			if !reflect.DeepEqual(actual, test.expected) {
				t.Log("H")
				t.Log(test.H)
				t.Fatalf("expected %v but found %v", test.expected, actual)
			}
		})
	}
}
