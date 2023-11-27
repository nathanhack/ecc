package linearblock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nathanhack/ecc/linearblock/internal"
	"github.com/nathanhack/ecc/linearblock/messagepassing/bec"
	mat "github.com/nathanhack/sparsemat"
	"github.com/sirupsen/logrus"
)

type Systematic struct {
	HColumnOrder []int
	G            mat.SparseMat
}

// LinearBlock contains matrices for the original H matrix and the systematic G generator.
type LinearBlock struct {
	H          mat.SparseMat //the original H(parity) matrix
	Processing *Systematic   // contains systematic generator matrix
}

// // For JSON unmarshalling
type systematic struct {
	HColumnOrder []int
	G            mat.CSRMatrix
}
type linearblock struct {
	H          mat.CSRMatrix
	Processing *systematic
}

// UnmarshalJSON is needed because LinearBlock has a mat.SparseMat and requires special handling
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

	l.Processing = &Systematic{
		HColumnOrder: lb.Processing.HColumnOrder,
		G:            &lb.Processing.G,
	}

	return nil
}

// Encode take in a message and encodes it using the linear block, returning a codeword
func (l *LinearBlock) Encode(message mat.SparseVector) (codeword mat.SparseVector) {
	G := l.Processing.G
	rows, cols := G.Dims()
	if message.Len() != rows {
		panic(fmt.Sprintf("message length == %v is required but found %v", rows, message.Len()))
	}

	codeword = mat.DOKVec(cols)
	codeword.MulMat(message, G)

	return ToNonSystematic(codeword, l.Processing.HColumnOrder)
}

// EncodeBE is encode for Binary Erasure channels
func (l *LinearBlock) EncodeBE(message mat.SparseVector) (codeword []bec.ErasureBit) {
	tmp := l.Encode(message)

	codeword = make([]bec.ErasureBit, tmp.Len())
	for i := 0; i < tmp.Len(); i++ {
		codeword[i] = bec.ErasureBit(tmp.At(i))
	}
	return codeword
}

// Decode takes in a codeword and returns the message contained in it
func (l *LinearBlock) Decode(codeword mat.SparseVector) (message mat.SparseVector) {
	if codeword.Len() != l.CodewordLength() {
		panic(fmt.Sprintf("codeword length == %v required but found %v", l.CodewordLength(), codeword.Len()))
	}

	ml := l.MessageLength()

	codeword = ToSystematic(codeword, l.Processing.HColumnOrder)
	return codeword.Slice(0, ml)
}

func (l *LinearBlock) DecodeBE(codeword []bec.ErasureBit) (message []bec.ErasureBit) {
	if len(codeword) != l.CodewordLength() {
		panic(fmt.Sprintf("codeword length == %v required but found %v", l.CodewordLength(), len(codeword)))
	}

	ml := l.MessageLength()

	codeword = ToSystematicBE(codeword, l.Processing.HColumnOrder)
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
	k, n := l.Processing.G.Dims()
	return n - k
}
func (l *LinearBlock) CodewordLength() int {
	_, n := l.Processing.G.Dims()
	return n
}
func (l *LinearBlock) CodeRate() float64 {
	return float64(l.MessageLength()) / float64(l.CodewordLength())
}

// Validate will test if this linearblock satisfies G*H.T=0, where G is the generator matrix and H.T is the transpose of H
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

// ToNonSystematic take in a systematic codeword and the ordering, it returns the nonsystematic form of it
func ToNonSystematic(codeword mat.SparseVector, ordering []int) mat.SparseVector {
	if len(ordering) > 0 && codeword.Len() != len(ordering) {
		panic("vector length must equal ordering length")
	}

	result := mat.DOKVec(codeword.Len())
	for c, c1 := range ordering {
		result.Set(c1, codeword.At(c))
	}

	return result
}

// ToNonSystematicBE take in a systematic codeword and the ordering, it returns the nonsystematic form of it
func ToNonSystematicBE(codeword []bec.ErasureBit, ordering []int) []bec.ErasureBit {
	if len(ordering) > 0 && len(codeword) != len(ordering) {
		panic("vector length must equal ordering length")
	}

	result := make([]bec.ErasureBit, len(codeword))
	for c, c1 := range ordering {
		result[c1] = codeword[c]
	}

	return result
}

// ToSystematic take in a nonsystematic codeword and the ordering, it returns the systematic form
func ToSystematic(codeword mat.SparseVector, ordering []int) mat.SparseVector {
	if len(ordering) > 0 && codeword.Len() != len(ordering) {
		panic("vector length must equal ordering length")
	}

	result := mat.DOKVec(codeword.Len())
	for c, c1 := range ordering {
		result.Set(c, codeword.At(c1))
	}

	return result
}

// ToSystematicBE take in a nonsystematic codeword and the ordering, it returns the systematic form
func ToSystematicBE(codeword []bec.ErasureBit, ordering []int) []bec.ErasureBit {
	if len(ordering) == 0 {
		panic("ordering length must be >0")
	}
	if len(codeword) != len(ordering) {
		panic("vector length must equal ordering length")
	}

	result := make([]bec.ErasureBit, len(codeword))
	for c, c1 := range ordering {
		result[c] = codeword[c1]
	}

	return result
}

// NonsystematicSplit takes in a nonsystematic codeword and the linearblock, it returns the message and parity bits
func NonsystematicSplit(codeword mat.SparseVector, block *LinearBlock) (message, parity mat.SparseVector) {
	systematic := ToSystematic(codeword, block.Processing.HColumnOrder)

	return SystematicSplit(systematic, block)
}

// SystematicSplit takes in a systematic codeword and splits it into the message and parity bits
func SystematicSplit(codeword mat.SparseVector, block *LinearBlock) (message, parity mat.SparseVector) {
	if codeword.Len() != block.CodewordLength() {
		panic("codeword length must block's codeword length")
	}
	return codeword.Slice(0, block.MessageLength()), codeword.Slice(block.MessageLength(), block.ParitySymbols())
}

// NonsystematicBESplit takes in a nonsystematic codeword and splits it into the message and parity bits
func NonsystematicBESplit(codeword []bec.ErasureBit, block *LinearBlock) (message, parity []bec.ErasureBit) {
	systematic := ToSystematicBE(codeword, block.Processing.HColumnOrder)

	return SystematicBESplit(systematic, block)
}

// SystematicBESplit takes in a systematic codeword and splits it into the message and parity bits
func SystematicBESplit(codeword []bec.ErasureBit, block *LinearBlock) (message, parity []bec.ErasureBit) {
	if len(codeword) != block.CodewordLength() {
		panic("codeword length must block's codeword length")
	}
	return codeword[:block.MessageLength()], codeword[block.MessageLength():]
}

func ExtractAFromH(ctx context.Context, H mat.SparseMat, threads int) (A mat.SparseMat, columnOrdering []int) {

	gje, ordering := internal.GaussianJordanEliminationGF2(ctx, H, threads)

	m, N := gje.Dims()

	logrus.Debug("Validating Row Reduced Matrix ")
	//let's check if we got a [ I, * ] format
	actual := gje.Slice(0, 0, m, m)
	ident := mat.CSRIdentity(m)
	if !actual.Equals(ident) {
		logrus.Errorf("failed to create transform H matrix into [I,*]")
		return nil, nil
	}

	//we need to convert gje from [ I, A] to [ A, I] (while keeping track)
	// and then extract A

	// first the keeping track part
	columnOrdering = make([]int, len(ordering))
	copy(columnOrdering[0:N-m], ordering[m:N])
	copy(columnOrdering[N-m:N], ordering[0:m])

	logrus.Debug("Extracting A Matrix from Row Reduced Matrix")
	//finally extract the A
	A = gje.Slice(0, m, m, N-m)

	return A, columnOrdering
}

// CreateHGPair creates a derived parity check matrix and G generator matrix from the parity matrix H.
// Note: the LinearBlock parity matrix's columns may be swapped.
func CreateHGPair(ctx context.Context, H mat.SparseMat, threads int) (HColumnOrder []int, G mat.SparseMat, parityCheckMatrix mat.SparseMat) {
	hRows, hCols := H.Dims()
	if hRows >= hCols {
		panic(fmt.Sprintf("H matrix shape == (rows, cols) where rows < cols required found rows:%v >= cols:%v", hRows, hCols))
	}
	// So we now take the current H matrix
	// convert H=[*] -> H=[A,I]
	// then extract out the A and keep track of columnSwaps during it
	logrus.Debugf("Creating generator matrix from H matrix")
	A, columnSwaps := ExtractAFromH(ctx, H, threads)
	if A == nil {
		logrus.Debugf("Unable to create generator matrix from H")
		return nil, nil, nil
	}

	AT := A.T() // transpose of A
	atRows, atCols := AT.Dims()

	logrus.Debug("Creating Generator Matrix")
	//Next using A make G=[I, A^T] where A^T is the transpose of A
	G = mat.DOKMat(atRows, atRows+atCols)
	G.SetMatrix(mat.CSRIdentity(atRows), 0, 0)
	G.SetMatrix(AT, 0, atRows)

	aRows, aCols := A.Dims()
	if aRows == hRows {
		logrus.Debugf("Generator Matrix complete")
		return columnSwaps, G, H
	}

	//well the H matrix's ranks != null space
	// so we need to create a parityCheckMatrix H=[A,I]
	logrus.Debugf("H Parity Check Matrix Reconstitution")
	parityCheckMatrix = mat.DOKMat(aRows, aRows+aCols)
	parityCheckMatrix.SetMatrix(A, 0, 0)
	parityCheckMatrix.SetMatrix(mat.CSRIdentity(aRows), 0, aCols)

	//with this change the columnSwap no longer holds so we reset the swaps
	columnSwaps = make([]int, aRows+aCols)
	for i := range columnSwaps {
		columnSwaps[i] = i
	}

	logrus.Debugf("H Parity Check Matrix Reconstitution Complete")
	return columnSwaps, G, parityCheckMatrix
}

func SystematicLinearBlock(ctx context.Context, H mat.SparseMat, threads int) *LinearBlock {
	// at this point we have a state that is "completed"
	order, g, pcm := CreateHGPair(ctx, H, threads)
	if order == nil {
		return nil
	}

	return &LinearBlock{
		H: pcm,
		Processing: &Systematic{
			HColumnOrder: order,
			G:            g,
		},
	}
}
