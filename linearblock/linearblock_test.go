package linearblock

import (
	mat "github.com/nathanhack/sparsemat"
	"math/rand"
	"reflect"
	"testing"
)

func TestOrderUnorderVector(t *testing.T) {
	vec := mat.DOKVec(100)
	columns := make([]int, vec.Len())
	for i := 0; i < vec.Len(); i++ {
		vec.Set(i, rand.Intn(2))
		columns[i] = i
	}

	rand.Shuffle(len(columns), func(i, j int) {
		tmp := columns[i]
		columns[i] = columns[j]
		columns[j] = tmp
	})

	swapped := ToSystematic(vec, columns)

	actual := ToNonSystematic(swapped, columns)

	if !reflect.DeepEqual(vec, actual) {
		t.Fatalf("expected %v but found %v", vec, actual)
	}
}
