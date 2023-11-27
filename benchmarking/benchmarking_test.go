package benchmarking

import (
	"context"
	"fmt"
	"runtime"

	"github.com/nathanhack/ecc/linearblock/hamming"
	"github.com/nathanhack/ecc/linearblock/messagepassing/bec"
	"github.com/nathanhack/ecc/linearblock/messagepassing/bec/iterative"
	"github.com/nathanhack/ecc/linearblock/messagepassing/bitflipping/harddecision"
	mat "github.com/nathanhack/sparsemat"
	mat2 "gonum.org/v1/gonum/mat"
)

func ExampleBenchmarkBSC() {
	linearBlock, _ := hamming.New(context.Background(), 3, 0)

	createMessage := func(trial int) mat.SparseVector {
		t := trial % 64
		message := mat.CSRVec(4)
		for i := 0; i < 4; i++ {
			message.Set(i, (t&(1<<i))>>i)
		}
		return message
	}

	encode := func(message mat.SparseVector) (codeword mat.SparseVector) {
		return linearBlock.Encode(message)
	}

	channel := func(originalCodeword mat.SparseVector) (erroredCodeword mat.SparseVector) {
		//since hamming can fix only one bit wrong we'll just flip one bit per codeword
		return RandomFlipBitCount(originalCodeword, 1)
	}
	repair := func(originalCodeword, channelInducedCodeword mat.SparseVector) (fixed mat.SparseVector) {
		alg := &harddecision.Gallager{
			H: linearBlock.H,
		}

		return harddecision.BitFlipping(alg, linearBlock.H, channelInducedCodeword, 50)
	}

	metrics := func(originalMessage, originalCodeword, fixedChannelInducedCodeword mat.SparseVector) (percentFixedCodewordErrors, percentFixedMessageErrors, percentFixedParityErrors float64) {
		codewordErrors := originalCodeword.HammingDistance(fixedChannelInducedCodeword)
		message := linearBlock.Decode(fixedChannelInducedCodeword)
		messageErrors := message.HammingDistance(originalMessage)
		parityErrors := codewordErrors - messageErrors

		percentFixedCodewordErrors = float64(codewordErrors) / float64(linearBlock.CodewordLength())
		percentFixedMessageErrors = float64(messageErrors) / float64(linearBlock.MessageLength())
		percentFixedParityErrors = float64(parityErrors) / float64(linearBlock.ParitySymbols())
		return
	}

	checkpoint := func(updatedStats Stats) {}

	stats := BenchmarkBSC(context.Background(), 1, 1, createMessage, encode, channel, repair, metrics, checkpoint, false)

	fmt.Println("Bit Error Probability :", stats)
	//Output:
	// Bit Error Probability : {Codeword:0.00(+/-0.00), Message:0.00(+/-0.00), Parity:0.00(+/-0.00)}
}

func ExampleBenchmarkBEC() {
	linearBlock, _ := hamming.New(context.Background(), 3, 0)

	createMessage := func(trial int) mat.SparseVector {
		t := trial % 64
		message := mat.CSRVec(4)
		for i := 0; i < 4; i++ {
			message.Set(i, (t&(1<<i))>>i)
		}
		return message
	}
	encode := func(message mat.SparseVector) (codeword []bec.ErasureBit) {
		return BitsToErased(linearBlock.Encode(message))
	}

	channel := func(originalCodeword []bec.ErasureBit) (erroredCodeword []bec.ErasureBit) {
		//since hamming can fix only one bit wrong under BSC but for BEC this code can fix 2 errors!!
		return RandomEraseCount(originalCodeword, 2)
	}

	repair := func(originalCodeword, channelInducedCodeword []bec.ErasureBit) (fixed []bec.ErasureBit) {
		alg := &iterative.Simple{
			H: linearBlock.H,
		}
		return bec.Flipping(alg, channelInducedCodeword)
	}

	metrics := func(originalMessage mat.SparseVector, originalCodeword, fixedChannelInducedCodeword []bec.ErasureBit) (percentFixedCodewordErrors, percentFixedMessageErrors, percentFixedParityErrors float64) {
		codewordErrors := ErasedCount(fixedChannelInducedCodeword)
		message := linearBlock.DecodeBE(fixedChannelInducedCodeword)
		messageErrors := ErasedCount(message)
		parityErrors := codewordErrors - messageErrors

		percentFixedCodewordErrors = float64(codewordErrors) / float64(linearBlock.CodewordLength())
		percentFixedMessageErrors = float64(messageErrors) / float64(linearBlock.MessageLength())
		percentFixedParityErrors = float64(parityErrors) / float64(linearBlock.ParitySymbols())
		return
	}

	checkpoint := func(updatedStats Stats) {
	}

	stats := BenchmarkBEC(context.Background(), 10000, 1, createMessage, encode, channel, repair, metrics, checkpoint, false)

	fmt.Println("Bit Error Probability :", stats)
	//Output:
	// Bit Error Probability : {Codeword:0.00(+/-0.00), Message:0.00(+/-0.00), Parity:0.00(+/-0.00)}
}

func ExampleBenchmarkBPSK() {
	threads := runtime.NumCPU()
	linearBlock, _ := hamming.New(context.Background(), 3, threads)

	createMessage := func(trial int) mat.SparseVector {
		t := trial % 64
		message := mat.CSRVec(4)
		for i := 0; i < 4; i++ {
			message.Set(i, (t&(1<<i))>>i)
		}
		return message
	}

	encode := func(message mat.SparseVector) (codeword mat2.Vector) {
		return BitsToBPSK(linearBlock.Encode(message))
	}

	channel := func(codeword mat2.Vector) (channelInducedCodeword mat2.Vector) {
		//since hamming can fix only one bit wrong, should have zero errors around 2Eb but will sometimes fail do to rounding
		return RandomNoiseBPSK(codeword, 2.0)
	}

	repair := func(originalCodeword, channelInducedCodeword mat2.Vector) (codeword mat2.Vector) {
		//we're going to simulate a hard decision of >=0 is 1
		// and <0 will be 0 on the output codeword

		tmp := mat.CSRVec(channelInducedCodeword.Len())
		for i := 0; i < channelInducedCodeword.Len(); i++ {
			if channelInducedCodeword.AtVec(i) >= 0 {
				tmp.Set(i, 1)
			}
		}
		alg := &harddecision.Gallager{H: linearBlock.H}
		//next we'll do the simple Gallager hard decision bit flipping with a max of 20 iterations
		tmp = harddecision.BitFlipping(alg, linearBlock.H, tmp, 20)
		return BitsToBPSK(tmp)
	}

	metrics := func(message mat.SparseVector, originalCodeword, fixedChannelInducedCodeword mat2.Vector) (percentFixedCodewordErrors, percentFixedMessageErrors, percentFixedParityErrors float64) {
		codewordErrors := HammingDistanceBPSK(originalCodeword, fixedChannelInducedCodeword)
		decoded := linearBlock.Decode(BPSKToBits(fixedChannelInducedCodeword, 0))
		messageErrors := decoded.HammingDistance(message)
		parityErrors := codewordErrors - messageErrors

		percentFixedCodewordErrors = float64(codewordErrors) / float64(linearBlock.CodewordLength())
		percentFixedMessageErrors = float64(messageErrors) / float64(linearBlock.MessageLength())
		percentFixedParityErrors = float64(parityErrors) / float64(linearBlock.ParitySymbols())
		return
	}

	checkpoint := func(updatedStats Stats) {}

	stats := BenchmarkBPSK(context.Background(), 100_000, threads, createMessage, encode, channel, repair, metrics, checkpoint, false)

	fmt.Println("Bit Error Probability :", stats)
	//Output:
	// Bit Error Probability : {Codeword:0.00(+/-0.04), Message:0.00(+/-0.05), Parity:0.00(+/-0.05)}
}
