package benchmarking

import (
	"math"
	"math/rand"

	"github.com/nathanhack/ecc/linearblock/messagepassing/bec"
	mat "github.com/nathanhack/sparsemat"
	mat2 "gonum.org/v1/gonum/mat"
)

// RandomMessage creates a random message of length len.
func RandomMessage(len int) mat.SparseVector {
	message := mat.CSRVec(len)
	for i := 0; i < len; i++ {
		message.Set(i, rand.Intn(2))
	}
	return message
}

// RandomMessage creates a random message o lenght len with a hamming weight equal to onesCount
func RandomMessageOnesCount(len int, onesCount int) mat.SparseVector {
	message := mat.CSRVec(len)
	for message.HammingWeight() < onesCount {
		message.Set(rand.Intn(len), 1)
	}
	return message
}

// RandomFlipBitCount randomly flips min(numberOfBitsToFlip,len(input)) number of bits.
func RandomFlipBitCount(input mat.SparseVector, numberOfBitsToFlip int) mat.SparseVector {
	output := mat.CSRVecCopy(input)

	flip := make(map[int]bool)
	for len(flip) < numberOfBitsToFlip && len(flip) < input.Len() {
		flip[rand.Intn(input.Len())] = true
	}

	for i := range flip {
		output.Set(i, output.At(i)+1)
	}
	return output
}

// RandomErase creates a new slice of ErasureBits with some of them set to Erased given the probabilityOfErasure
func RandomErase(codeword []bec.ErasureBit, probabilityOfErasure float64) []bec.ErasureBit {
	return RandomEraseCount(codeword, int(math.Round(probabilityOfErasure*float64(len(codeword)))))
}

// RandomErase creates a copy of the codeword and randomly sets numberOfBitsToFlip of them to Erased
func RandomEraseCount(codeword []bec.ErasureBit, numberOfBitsToFlip int) []bec.ErasureBit {
	output := make([]bec.ErasureBit, len(codeword))

	//randomly pick indices to erase
	flip := make(map[int]bool)
	for len(flip) < numberOfBitsToFlip {
		flip[rand.Intn(len(codeword))] = true
	}

	//copy the old data
	for i := range codeword {
		output[i] = codeword[i]
	}

	//set the erased
	for i := range flip {
		output[i] = bec.Erased
	}

	return output
}

// RandomNoiseBPSK creates a randomizes version of the bpsk vector using the E_b/N_0 passed in
func RandomNoiseBPSK(bpsk mat2.Vector, E_bPerN_0 float64) mat2.Vector {
	//using  σ^2 = N_0/2 and E_b=1
	// we get  σ = sqrt(1/(2*E_bPerN_0))
	σ := math.Sqrt(1 / (2 * E_bPerN_0))
	result := mat2.NewVecDense(bpsk.Len(), nil)
	for i := 0; i < bpsk.Len(); i++ {
		result.SetVec(i, rand.NormFloat64()*σ)
	}
	result.AddVec(result, bpsk)
	return result
}
