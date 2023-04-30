package bec

import (
	"context"
	"math"
	"math/rand"
	"sync"

	"github.com/nathanhack/ecc/benchmarking"
	"github.com/nathanhack/ecc/linearblock"
	"github.com/nathanhack/ecc/linearblock/messagepassing/bec"
	mat "github.com/nathanhack/sparsemat"
)

const bitLimit = 64

func RunBEC(ctx context.Context,
	l *linearblock.LinearBlock,
	percentage float64, trials, threads int,
	correctionAlg benchmarking.BinaryErasureChannelCorrection,
	previousStats benchmarking.Stats,
	checkpoints benchmarking.Checkpoints,
	showProgressBar bool) benchmarking.Stats {
	messageHistory := make(map[string]bool)
	messageHistoryMux := sync.RWMutex{}
	messageHistoryMax := math.Pow(2, float64(l.MessageLength()))

	createMessage := func(trial int) mat.SparseVector {
		message := mat.CSRVec(l.MessageLength())
		messageHistoryMux.RLock()
		_, has := messageHistory[message.String()]
		messageHistoryMux.RUnlock()
		for has {
			reset := false
			for i := 0; i < l.MessageLength(); i++ {
				message.Set(i, rand.Intn(2))
			}
			messageHistoryMux.RLock()
			_, has = messageHistory[message.String()]
			reset = float64(len(messageHistory)) >= messageHistoryMax
			messageHistoryMux.RUnlock()

			if reset {
				messageHistoryMux.Lock()
				messageHistory = make(map[string]bool)
				messageHistoryMux.Unlock()
			}
		}
		messageHistoryMux.Lock()
		//when the message is relatively small we'll keep track so we don't have dups
		if message.Len() < bitLimit || message.IsZero() {
			messageHistory[message.String()] = true
		}
		messageHistoryMux.Unlock()
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
