package bec

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"math"

	"github.com/nathanhack/ecc/benchmarking"
	"github.com/nathanhack/ecc/linearblock"
	"github.com/nathanhack/ecc/linearblock/messagepassing/bec"
	mat "github.com/nathanhack/sparsemat"
)

const bitLimit = 30

func RunBEC(ctx context.Context,
	l *linearblock.LinearBlock,
	percentage float64, trials, threads int,
	correctionAlg benchmarking.BinaryErasureChannelCorrection,
	previousStats benchmarking.Stats,
	checkpoints benchmarking.Checkpoints,
	showProgressBar bool) benchmarking.Stats {

	createMessage := func(trial int) mat.SparseVector {
		message := mat.CSRVec(l.MessageLength())

		// if the size of the message is small enough we'll track everything
		if l.MessageLength() <= bitLimit {
			target := trial % ((1 << l.MessageLength()) - 1)
			for i := 0; i < l.MessageLength(); i++ {
				if target&1<<i > 0 {
					message.Set(i, 1)
				} else {
					message.Set(i, 0)
				}
			}
			return message
		}

		// if not small enough to track everything then we'll use some crypto rand
		// to make our messages
		bs := make([]byte, int(math.Ceil(float64(l.MessageLength())/8)))

		n, err := crand.Read(bs)
		if err != nil {
			panic(fmt.Sprintf("random message error: %v", err))
		} else if n != len(bs) {
			panic(fmt.Sprintf("random message expected %v found %v", len(bs), n))
		}

		for j, b := range bs {
			for i := 0; i < 8 && j*8+i < l.MessageLength(); i++ {
				message.Set(j*8+i, int((b>>i)&1))
			}
		}

		return message
	}

	encode := func(message mat.SparseVector) (codeword []bec.ErasureBit) {
		return l.EncodeBE(message)
	}

	channel := func(originalCodeword []bec.ErasureBit) (erroredCodeword []bec.ErasureBit) {
		count := int(percentage * float64(len(originalCodeword)))
		return benchmarking.RandomEraseCount(originalCodeword, count)
	}

	metrics := func(originalMessage mat.SparseVector, originalCodeword, fixedChannelInducedCodeword []bec.ErasureBit) (percentFixedCodewordErrors, percentFixedMessageErrors, percentFixedParityErrors float64) {
		codewordErrors := benchmarking.ErasedCount(fixedChannelInducedCodeword)
		message := l.DecodeBE(fixedChannelInducedCodeword)
		messageErrors := benchmarking.ErasedCount(message)
		parityErrors := codewordErrors - messageErrors

		percentFixedCodewordErrors = float64(codewordErrors) / float64(l.CodewordLength())
		percentFixedMessageErrors = float64(messageErrors) / float64(l.MessageLength())
		percentFixedParityErrors = float64(parityErrors) / float64(l.ParitySymbols())
		return
	}

	return benchmarking.BenchmarkBECContinueStats(ctx, trials, threads, createMessage, encode, channel, correctionAlg, metrics, checkpoints, previousStats, showProgressBar)
}
