package benchmarking

import (
	"context"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/nathanhack/avgstd"
	"github.com/nathanhack/errorcorrectingcodes/linearblock/messagepassing/bec"
	mat "github.com/nathanhack/sparsemat"
	"github.com/nathanhack/threadpool"
	mat2 "gonum.org/v1/gonum/mat"
	"math"
	"sync"
)

type Stats struct {
	ChannelCodewordError avgstd.AvgStd // probability of a bit error after channel errors are fixed
	ChannelMessageError  avgstd.AvgStd // probability of a bit error after channel errors are fixed
	ChannelParityError   avgstd.AvgStd // probability of a bit error after channel errors are fixed
}

func (s Stats) String() string {
	return fmt.Sprintf("{Codeword:%0.02f(+/-%0.02f), Message:%0.02f(+/-%0.02f), Parity:%0.02f(+/-%0.02f)}",
		s.ChannelCodewordError.Mean, math.Sqrt(s.ChannelCodewordError.SampledVariance()),
		s.ChannelMessageError.Mean, math.Sqrt(s.ChannelMessageError.SampledVariance()),
		s.ChannelParityError.Mean, math.Sqrt(s.ChannelParityError.SampledVariance()),
	)
}

type Checkpoints func(updatedStats Stats)

type BinaryMessageConstructor func(trial int) (message mat.SparseVector)

//specfic to BSC
type BinarySymmetricChannelEncoder func(message mat.SparseVector) (codeword mat.SparseVector)
type BinarySymmetricChannel func(codeword mat.SparseVector) (channelInducedCodeword mat.SparseVector)
type BinarySymmetricChannelCorrection func(originalCodeword, channelInducedCodeword mat.SparseVector) (fixedChannelInducedCodeword mat.SparseVector)
type BinarySymmetricChannelMetrics func(originalMessage, originalCodeword, fixedChannelInducedCodeword mat.SparseVector) (percentFixedCodewordErrors, percentFixedMessageErrors, percentFixedParityErrors float64)

//specific to BEC
type BinaryErasureChannelEncoder func(message mat.SparseVector) (codeword []bec.ErasureBit)
type BinaryErasureChannel func(codeword []bec.ErasureBit) (channelInducedCodeword []bec.ErasureBit)
type BinaryErasureChannelCorrection func(originalCodeword, channelInducedCodeword []bec.ErasureBit) (fixedChannelInducedCodeword []bec.ErasureBit)
type BinaryErasureChannelMetrics func(originalMessage mat.SparseVector, originalCodeword, fixedChannelInducedCodeword []bec.ErasureBit) (percentFixedCodewordErrors, percentFixedMessageErrors, percentFixedParityErrors float64)

//specific to BPSK
type BPSKChannelEncoder func(message mat.SparseVector) (codeword mat2.Vector)
type BPSKChannel func(codeword mat2.Vector) (channelInducedCodeword mat2.Vector)
type BPSKChannelCorrection func(originalCodeword, channelInducedCodeword mat2.Vector) (fixedChannelInducedCodeword mat2.Vector)
type BPSKChannelMetrics func(originalMessage mat.SparseVector, originalCodeword, fixedChannelInducedCodeword mat2.Vector) (percentFixedCodewordErrors, percentFixedMessageErrors, percentFixedParityErrors float64)

func BenchmarkBSC(ctx context.Context,
	trials int, threads int,
	createMessage BinaryMessageConstructor,
	encode BinarySymmetricChannelEncoder,
	channel BinarySymmetricChannel,
	codewordRepair BinarySymmetricChannelCorrection,
	metrics BinarySymmetricChannelMetrics,
	checkpoints Checkpoints,
	showProgress bool) Stats {
	return BenchmarkBSCContinueStats(ctx, trials, threads, createMessage, encode, channel, codewordRepair, metrics, checkpoints, Stats{}, showProgress)
}

func BenchmarkBSCContinueStats(ctx context.Context,
	trials int, threads int,
	createMessage BinaryMessageConstructor,
	encode BinarySymmetricChannelEncoder,
	channel BinarySymmetricChannel,
	codewordRepair BinarySymmetricChannelCorrection,
	metrics BinarySymmetricChannelMetrics,
	checkpoints Checkpoints,
	previousStats Stats,
	showProgress bool) Stats {
	trialsToRun := trials - previousStats.ChannelCodewordError.Count
	if trialsToRun <= 0 {
		return previousStats
	}

	var bar *pb.ProgressBar
	if showProgress {
		bar = pb.StartNew(trialsToRun)
	}

	pool := threadpool.New(ctx, threads, trialsToRun)
	statsMux := sync.Mutex{}

	trial := func(i int) {
		if showProgress {
			bar.Increment()
		}
		//we create a random message
		message := createMessage(i)

		// encode to get our codeword
		codeword := encode(message)

		// send through the channel to get channel induced errors
		channelInducedCodeword := channel(codeword)

		// repair the codeword (if possible)
		repaired := codewordRepair(codeword, channelInducedCodeword)

		// get metrics
		percentFixedCodewordErrors, percentFixedMessageErrors, percentFixedParityErrors := metrics(message, codeword, repaired)

		statsMux.Lock()
		previousStats.ChannelCodewordError.Update(percentFixedCodewordErrors)
		previousStats.ChannelMessageError.Update(percentFixedMessageErrors)
		previousStats.ChannelParityError.Update(percentFixedParityErrors)
		if checkpoints != nil {
			checkpoints(previousStats) //give them the updated checkpoint
		}
		statsMux.Unlock()
	}

	for i := previousStats.ChannelCodewordError.Count; i < trials; i++ {
		tmp := i
		pool.Add(func() { trial(tmp) })
	}
	pool.Wait()
	if showProgress {
		bar.Finish()
	}
	return previousStats
}

func BenchmarkBEC(ctx context.Context,
	trials, threads int,
	createMessage BinaryMessageConstructor,
	encode BinaryErasureChannelEncoder,
	channel BinaryErasureChannel,
	codewordRepair BinaryErasureChannelCorrection,
	metrics BinaryErasureChannelMetrics,
	checkpoints Checkpoints, showBar bool) Stats {
	return BenchmarkBECContinueStats(ctx, trials, threads, createMessage, encode, channel, codewordRepair, metrics, checkpoints, Stats{}, showBar)
}

func BenchmarkBECContinueStats(
	ctx context.Context,
	trials, threads int,
	createMessage BinaryMessageConstructor,
	encode BinaryErasureChannelEncoder,
	channel BinaryErasureChannel,
	codewordRepair BinaryErasureChannelCorrection,
	metrics BinaryErasureChannelMetrics,
	checkpoints Checkpoints,
	previousStats Stats,
	showProgressBar bool) Stats {
	trialsToRun := trials - previousStats.ChannelCodewordError.Count
	if trialsToRun <= 0 {
		return previousStats
	}

	var bar *pb.ProgressBar
	if showProgressBar {
		bar = pb.StartNew(trialsToRun)
	}

	pool := threadpool.New(ctx, threads, trialsToRun)
	statsMux := sync.Mutex{}

	trial := func(i int) {
		if showProgressBar {
			bar.Increment()
		}
		//we create a random message
		message := createMessage(i)

		// encode to get our codeword
		codeword := encode(message)

		// send through the channel to get channel induced errors
		channelInducedCodeword := channel(codeword)

		// repair the codeword (if possible) and return metrics
		repaired := codewordRepair(codeword, channelInducedCodeword)

		// get metrics
		percentFixedCodewordErrors, percentFixedMessageErrors, percentFixedParityErrors := metrics(message, codeword, repaired)

		statsMux.Lock()
		previousStats.ChannelCodewordError.Update(percentFixedCodewordErrors)
		previousStats.ChannelMessageError.Update(percentFixedMessageErrors)
		previousStats.ChannelParityError.Update(percentFixedParityErrors)

		if checkpoints != nil {
			checkpoints(previousStats) //give them the updated checkpoint
		}
		statsMux.Unlock()
	}

	for i := previousStats.ChannelCodewordError.Count; i < trials; i++ {
		pool.Add(func() { trial(i) })
	}

	pool.Wait()
	if showProgressBar {
		bar.Finish()
	}
	return previousStats
}

func BenchmarkBPSK(ctx context.Context,
	trials int, threads int,
	createMessage BinaryMessageConstructor,
	encode BPSKChannelEncoder,
	channel BPSKChannel,
	codewordRepair BPSKChannelCorrection,
	metrics BPSKChannelMetrics,
	checkpoints Checkpoints, showProgress bool) Stats {
	return BenchmarkBPSKContinueStats(ctx, trials, threads, createMessage, encode, channel, codewordRepair, metrics, checkpoints, Stats{}, showProgress)
}

func BenchmarkBPSKContinueStats(ctx context.Context,
	trials int, threads int,
	createMessage BinaryMessageConstructor,
	encode BPSKChannelEncoder,
	channel BPSKChannel,
	codewordRepair BPSKChannelCorrection,
	metrics BPSKChannelMetrics,
	checkpoints Checkpoints,
	previousStats Stats,
	showProgress bool) Stats {
	trialsToRun := trials - previousStats.ChannelCodewordError.Count
	if trialsToRun <= 0 {
		return previousStats
	}

	var bar *pb.ProgressBar
	if showProgress {
		bar = pb.StartNew(trialsToRun)
	}
	pool := threadpool.New(ctx, threads, trialsToRun)
	statsMux := sync.Mutex{}

	trial := func(i int) {
		if showProgress {
			bar.Increment()
		}
		//we create a random message
		message := createMessage(i)

		// encode to get our codeword
		codeword := encode(message)

		// send through the channel to get channel induced errors
		channelInducedCodeword := channel(codeword)

		// repair the codeword (if possible)
		repaired := codewordRepair(codeword, channelInducedCodeword)

		// get metrics
		percentFixedCodewordErrors, percentFixedMessageErrors, percentFixedParityErrors := metrics(message, codeword, repaired)

		statsMux.Lock()
		previousStats.ChannelCodewordError.Update(percentFixedCodewordErrors)
		previousStats.ChannelMessageError.Update(percentFixedMessageErrors)
		previousStats.ChannelParityError.Update(percentFixedParityErrors)

		if checkpoints != nil {
			checkpoints(previousStats) //give them the updated checkpoint
		}
		statsMux.Unlock()
	}

	for i := previousStats.ChannelCodewordError.Count; i < trials; i++ {
		pool.Add(func() { trial(i) })
	}
	pool.Wait()
	if showProgress {
		bar.Finish()
	}
	return previousStats
}

//HammingDistanceErasuresToBits calculates number of bits different.
// If a and b are different sizes it assumes they are
// both aligned with the zero index (the difference is at the end)
func HammingDistanceErasuresToBits(a []bec.ErasureBit, b mat.SparseVector) int {
	min := len(a)
	max := b.Len()
	if min > max {
		min = b.Len()
		max = len(a)
	}

	count := 0
	for i := 0; i < min; i++ {
		if int(a[i]) != b.At(i) {
			count++
		}
	}
	return max - min + count
}

//BitsToBPSK converts a [0,1] matrix to a [-1,1] matrix
func BitsToBPSK(a mat.SparseVector) mat2.Vector {
	output := mat2.NewVecDense(a.Len(), nil)

	for i := 0; i < a.Len(); i++ {
		if a.At(i) > 0 {
			output.SetVec(i, 1)
		} else {
			output.SetVec(i, -1)
		}
	}

	return output
}

//BPSKToBits conversts a BPSK vector [-1,1] to sparse vector [0,1].
// Values >= boundary will be considered a 1, otherwise a 0.
func BPSKToBits(a mat2.Vector, boundary float64) mat.SparseVector {
	result := mat.CSRVec(a.Len())

	for i := 0; i < a.Len(); i++ {
		if a.AtVec(i) >= boundary {
			result.Set(i, 1)
		}
	}
	return result
}

//HammingDistanceBPSK calculates number of bits different.
// Assumes >=0 is 1 and <0 is 0
// If a and b are different sizes it assumes they are
// both aligned with the zero index (the difference is at the end)
func HammingDistanceBPSK(a, b mat2.Vector) int {
	min := a.Len()
	max := b.Len()
	if min > max {
		min = b.Len()
		max = a.Len()
	}

	count := 0
	for i := 0; i < min; i++ {
		aOne := a.AtVec(i) >= 0
		bOne := b.AtVec(i) >= 0
		if aOne != bOne {
			count++
		}
	}
	return max - min + count
}

//BitsToErased creates a slice of ErasureBits from the codeword passed in
func BitsToErased(codeword mat.SparseVector) []bec.ErasureBit {
	output := make([]bec.ErasureBit, codeword.Len())
	for i := 0; i < codeword.Len(); i++ {
		output[i] = bec.ErasureBit(codeword.At(i))
	}
	return output
}

//ErasedCount returns the number of erased bits
func ErasedCount(base []bec.ErasureBit) (count int) {
	for _, e := range base {
		if e == bec.Erased {
			count++
		}
	}
	return
}
