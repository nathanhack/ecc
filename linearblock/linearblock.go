package linearblock

import (
	"encoding/json"
	"fmt"
	"github.com/nathanhack/errorcorrectingcodes/linearblock/internal"
	"github.com/nathanhack/errorcorrectingcodes/linearblock/messagepassing/bec"
	mat "github.com/nathanhack/sparsemat"
	"strings"
)

type Systemic struct {
	HColumnOrder []int
	G            mat.SparseMat
}

//LinearBlock contains matrices for the original H matrix and the systemic G generator.
type LinearBlock struct {
	H          mat.SparseMat //the original H(parity) matrix
	Processing *Systemic     // contains systemic generator matrix
}

//// For JSON unmarshalling
type systemic struct {
	HColumnOrder []int
	G            mat.CSRMatrix
}
type linearblock struct {
	H          mat.CSRMatrix
	Processing *systemic
}

//UnmarshalJSON is needed because LinearBlock has a mat.SparseMat and requires special handling
func (l *LinearBlock) UnmarshalJSON(bytes []byte) error {
	var lb linearblock
	err := json.Unmarshal(bytes, &lb)
	if err != nil {
		return err
	}

	l.H = &lb.H
	if lb.Processing == nil {
		return nil
	}

	l.Processing = &Systemic{
		HColumnOrder: lb.Processing.HColumnOrder,
		G:            &lb.Processing.G,
	}

	return nil
}

//Encode take in a message and encodes it using the linear block, returning a codeword
func (l *LinearBlock) Encode(message mat.SparseVector) (codeword mat.SparseVector) {
	G := l.Processing.G
	rows, cols := G.Dims()
	if message.Len() != rows {
		panic(fmt.Sprintf("message length == %v is required but found %v", rows, message.Len()))
	}

	codeword = mat.DOKVec(cols)
	codeword.MulMat(message, G)

	return unorderVector(codeword, l.Processing.HColumnOrder)
}

//EncodeBE is encode for Binary Erasure channels
func (l *LinearBlock) EncodeBE(message mat.SparseVector) (codeword []bec.ErasureBit) {
	tmp := l.Encode(message)

	codeword = make([]bec.ErasureBit, tmp.Len())
	for i := 0; i < tmp.Len(); i++ {
		codeword[i] = bec.ErasureBit(tmp.At(i))
	}
	return codeword
}

func unorderVector(codeword mat.SparseVector, ordering []int) mat.SparseVector {
	if len(ordering) > 0 && codeword.Len() != len(ordering) {
		panic("vector length must equal ordering length")
	}
	result := mat.DOKVec(codeword.Len())

	for c, c1 := range ordering {
		result.Set(c1, codeword.At(c))
	}

	return result
}

func orderVector(codeword mat.SparseVector, ordering []int) mat.SparseVector {
	if len(ordering) > 0 && codeword.Len() != len(ordering) {
		panic("vector length must equal ordering length")
	}
	result := mat.DOKVec(codeword.Len())

	for c, c1 := range ordering {
		result.Set(c, codeword.At(c1))
	}

	return result
}

func orderVectorBE(codeword []bec.ErasureBit, ordering []int) []bec.ErasureBit {
	if len(ordering) == 0 {
		panic("ordering length must be >0")
	}
	if len(ordering) > 0 && len(codeword) != len(ordering) {
		panic("vector length must equal ordering length")
	}
	result := make([]bec.ErasureBit, len(codeword))

	for c, c1 := range ordering {
		result[c] = codeword[c1]
	}

	return result
}

//Decode takes in a codeword and returns the message contained in it
func (l *LinearBlock) Decode(codeword mat.SparseVector) (message mat.SparseVector) {
	if codeword.Len() != l.CodewordLength() {
		panic(fmt.Sprintf("codeword length == %v required but found %v", l.CodewordLength(), codeword.Len()))
	}

	ml := l.MessageLength()

	codeword = orderVector(codeword, l.Processing.HColumnOrder)
	return codeword.Slice(0, ml)
}

func (l *LinearBlock) DecodeBE(codeword []bec.ErasureBit) (message []bec.ErasureBit) {
	if len(codeword) != l.CodewordLength() {
		panic(fmt.Sprintf("codeword length == %v required but found %v", l.CodewordLength(), len(codeword)))
	}

	ml := l.MessageLength()

	codeword = orderVectorBE(codeword, l.Processing.HColumnOrder)
	return codeword[0:ml]
}

func (l *LinearBlock) Syndrome(codeword mat.SparseVector) (syndrome mat.SparseVector) {
	syndrome = mat.CSRVec(l.ParitySymbols())
	syndrome.MatMul(l.H, codeword)
	return
}

func (l *LinearBlock) MessageLength() int {
	k, _ := l.Processing.G.Dims()
	return k
}
func (l *LinearBlock) ParitySymbols() int {
	m, _ := l.H.Dims()
	return m
}
func (l *LinearBlock) CodewordLength() int {
	_, n := l.H.Dims()
	return n
}
func (l *LinearBlock) CodeRate() float64 {
	return float64(l.MessageLength()) / float64(l.CodewordLength())
}

//Validate will test if this linearblock satisfies G*H.T=0, where G is the generator matrix and H.T is the transpose of H
func (l *LinearBlock) Validate() bool {
	//now we validate it
	return internal.ValidateHGMatrices(l.Processing.G, internal.ColumnSwapped(l.H, l.Processing.HColumnOrder))
}

func (l *LinearBlock) String() string {
	buf := strings.Builder{}
	buf.WriteString("{\nH:\n")
	buf.WriteString(l.H.String())
	buf.WriteString(fmt.Sprintf("Order: %v", l.Processing.HColumnOrder))
	buf.WriteString("\nG:\n")
	buf.WriteString(l.Processing.G.String())
	buf.WriteString("\n}\n")
	return buf.String()
}
