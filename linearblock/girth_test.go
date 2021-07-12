package linearblock

import (
	mat "github.com/nathanhack/sparsemat"
	"strconv"
	"testing"
)

func TestCalculateGirthLowerBound(t *testing.T) {
	tests := []struct {
		h        mat.SparseMat
		minGirth int
		expected int
	}{
		{mat.CSRIdentity(500), -1, -1},
		{mat.CSRIdentity(500), 4, -1},
		{mat.CSRMat(2, 2, 1, 1, 1, 1), -1, 4},
		{mat.CSRMat(2, 2, 1, 1, 1, 1), 6, 4},
		{mat.CSRMat(2, 2, 1, 0, 0, 1), -1, -1},
		{mat.CSRMat(2, 2, 1, 0, 0, 1), 4, -1},
		{mat.CSRMat(4, 8, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 1, 0, 0, 0, 0, 0, 1, 1), -1, 8},
		{mat.CSRMat(3, 6, 1, 1, 1, 0, 0, 0, 0, 0, 1, 1, 1, 0, 1, 0, 0, 0, 1, 1), -1, 6},
		{mat.CSRMat(3, 6, 1, 1, 1, 0, 0, 0, 0, 0, 1, 1, 1, 0, 1, 0, 0, 0, 1, 1), 6, 6},
		{mat.CSRMat(3, 6, 1, 1, 1, 0, 0, 0, 0, 0, 1, 1, 1, 0, 1, 0, 0, 0, 1, 1), 4, -1},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actual := CalculateGirthLowerBound(test.h, test.minGirth, -1)
			if actual != test.expected {
				t.Fatalf("expected %v but found %v", test.expected, actual)
			}
		})
	}
}
func TestCalculateCycleLowerBound(t *testing.T) {
	tests := []struct {
		h          mat.SparseMat
		checkIndex int
		minGirth   int
		expected   int
	}{
		{mat.CSRMat(2, 2, 1, 1, 1, 1), 0, -1, 4},
		{mat.CSRMat(2, 2, 1, 1, 1, 1), 0, 6, 4},
		{mat.CSRMat(2, 2, 1, 0, 0, 1), 0, -1, -1},
		{mat.CSRMat(2, 2, 1, 0, 0, 1), 0, 4, -1},
		{mat.CSRMat(2, 2, 1, 0, 1, 0), 0, -1, -1},
		{mat.CSRMat(2, 2, 1, 0, 1, 0), 0, 4, -1},
		{mat.CSRIdentity(500), 0, -1, -1},
		{mat.CSRIdentity(500), 0, 4, -1},
		{mat.CSRMat(4, 8, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 1, 0, 0, 0, 0, 0, 1, 1), 0, -1, 8},
		{mat.CSRMat(3, 6, 1, 1, 1, 0, 0, 0, 0, 0, 1, 1, 1, 0, 1, 0, 0, 0, 1, 1), 0, -1, 6},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actual := CalculateCycleLowerBound(test.h, test.checkIndex, test.minGirth)
			if actual != test.expected {
				t.Fatalf("expected %v but found %v", test.expected, actual)
			}
		})
	}
}
func BenchmarkCalculateGirthLowerBound(b *testing.B) {
	h := mat.CSRMat(2, 2, 1, 1, 1, 1)
	for i := 0; i < b.N; i++ {
		CalculateGirthLowerBound(h, -1, 1)
	}
}

func BenchmarkCalculateGirthLowerBound2(b *testing.B) {
	h := mat.CSRIdentity(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateGirthLowerBound(h, -1, 0)
	}
}
func BenchmarkCalculateGirthLowerBound3(b *testing.B) {
	h := mat.CSRIdentity(10000)
	for i := 0; i < 10000-2; i++ {
		h.SetMatrix(mat.CSRMat(2, 2, 1, 1, 1, 1), i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateGirthLowerBound(h, -1, 0)
	}
}
