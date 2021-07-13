package bsc

import (
	"context"
	"github.com/nathanhack/errorcorrectingcodes/benchmarking"
	"github.com/nathanhack/errorcorrectingcodes/linearblock"
	mat "github.com/nathanhack/sparsemat"
	"math"
	"math/rand"
	"sync"
)

const bitLimit = 64

func RunBSC(ctx context.Context,
	l *linearblock.LinearBlock,
	crossoverProbability float64, trials, threads int,
	correctionAlg benchmarking.BinarySymmetricChannelCorrection,
	previousStats benchmarking.Stats,
	checkpoints benchmarking.Checkpoints,
	showProgress bool) benchmarking.Stats {
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
		if message.Len() < bitLimit || message.IsZero() {
			messageHistory[message.String()] = true
		}
		messageHistoryMux.Unlock()
		return message
	}

	encode := func(message mat.SparseVector) (codeword mat.SparseVector) {
		return l.Encode(message)
	}

	channel := func(originalCodeword mat.SparseVector) (erroredCodeword mat.SparseVector) {
		count := int(crossoverProbability * float64(originalCodeword.Len()))
		return benchmarking.RandomFlipBitCount(originalCodeword, count)
	}

	metrics := func(originalMessage, originalCodeword, fixedChannelInducedCodeword mat.SparseVector) (percentFixedCodewordErrors, percentFixedMessageErrors, percentFixedParityErrors float64) {
		codewordErrors := originalCodeword.HammingDistance(fixedChannelInducedCodeword)
		message := l.Decode(fixedChannelInducedCodeword)
		messageErrors := message.HammingDistance(originalMessage)
		parityErrors := codewordErrors - messageErrors

		percentFixedCodewordErrors = float64(codewordErrors) / float64(l.CodewordLength())
		percentFixedMessageErrors = float64(messageErrors) / float64(l.MessageLength())
		percentFixedParityErrors = float64(parityErrors) / float64(l.ParitySymbols())
		return
	}

	return benchmarking.BenchmarkBSCContinueStats(ctx, trials, threads, createMessage, encode, channel, correctionAlg, metrics, checkpoints, previousStats, showProgress)
}
