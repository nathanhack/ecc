package internal

// func TestExtractAFromH(t *testing.T) {
// 	tests := []struct {
// 		input    mat.SparseMat
// 		expected mat.SparseMat
// 	}{
// 		{ //Hamming 7
// 			mat.CSRMat(3, 7, 1, 0, 0, 1, 1, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 1),
// 			mat.CSRMat(3, 4, 1, 1, 1, 0, 1, 1, 0, 1, 0, 1, 1, 1),
// 		},
// 		{ //Random - one linearly dependent row
// 			mat.CSRMat(4, 5, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 1, 0, 1, 0, 0, 0, 0, 0, 1, 1),
// 			nil,
// 		},
// 	}
// 	for i, test := range tests {
// 		t.Run(strconv.Itoa(i), func(t *testing.T) {

// 			A, _ := ExtractAFromH(context.Background(), test.input, 0)

// 			if test.expected != nil {
// 				if !test.expected.Equals(A) {
// 					t.Fatalf("expected \n%v\n but found \n%v\n", test.expected, A)
// 				}
// 			} else {
// 				if A != nil {
// 					t.Fatalf("expecte nil but found \n%v\n", A)
// 				}
// 			}
// 		})
// 	}
// }
