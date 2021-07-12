package bec

type ErasureBit int

const (
	Zero ErasureBit = iota
	One
	Erased
)

type BECFlippingAlg interface {
	Flip(currentCodeword []ErasureBit) (nextCodeword []ErasureBit, done bool)
}

func Flipping(alg BECFlippingAlg, codeword []ErasureBit) (result []ErasureBit) {
	done := false
	result = make([]ErasureBit, len(codeword))
	for i := 0; i < len(codeword); i++ {
		result[i] = codeword[i]
	}
	for !done {
		result, done = alg.Flip(result)
	}
	return result
}
