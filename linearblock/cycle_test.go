package linearblock

import (
	mat "github.com/nathanhack/sparsemat"
	"strconv"
	"testing"
)

func TestSmallestCycle(t *testing.T) {
	tests := []struct {
		h        mat.SparseMat
		expected Cycle
	}{
		{mat.CSRMat(2, 2, 1, 1, 1, 1), Cycle{
			{Index: 0, Check: true},
			{Index: 0, Check: false},
			{Index: 1, Check: true},
			{Index: 1, Check: false},
		}},
		{mat.CSRMat(3, 3, 1, 1, 0, 0, 1, 1, 1, 0, 1), Cycle{
			{Index: 0, Check: true},
			{Index: 0, Check: false},
			{Index: 2, Check: true},
			{Index: 2, Check: false},
			{Index: 1, Check: true},
			{Index: 1, Check: false},
		}},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actual := SmallestCycle(test.h, false)
			if !actual.Equal(test.expected) {
				t.Fatalf("expected %v but found %v", test.expected, actual)
			}
		})
	}
}

func TestCycle_Equal(t *testing.T) {

	c1 := Cycle{
		{Index: 0, Check: true},
		{Index: 0, Check: false},
		{Index: 1, Check: true},
		{Index: 1, Check: false},
	}
	c2 := Cycle{
		{Index: 0, Check: true},
		{Index: 0, Check: false},
		{Index: 1, Check: true},
		{Index: 1, Check: false},
	}
	c3 := Cycle{
		{Index: 0, Check: true},
		{Index: 1, Check: false},
		{Index: 1, Check: true},
		{Index: 0, Check: false},
	}

	if !c1.Equal(c2) {
		t.Fatalf("expected equal")
	}

	if !c1.Equal(c3) {
		t.Fatalf("expected equal")
	}
}
