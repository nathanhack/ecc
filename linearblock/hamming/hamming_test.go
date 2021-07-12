package hamming

import (
	"context"
	"strconv"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		paritySymbols int
	}{
		{3},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actual, err := New(context.Background(), test.paritySymbols, 0)
			if err != nil {
				t.Fatalf("expected no error found :%v", err)
			}

			if !actual.Validate() {
				t.Fatalf("expected valid linearblock code")
			}
		})
	}
}
