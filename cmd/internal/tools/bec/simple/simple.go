package simple

import (
	"context"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/nathanhack/errorcorrectingcodes/benchmarking"
	"github.com/nathanhack/errorcorrectingcodes/cmd/internal/tools"
	"github.com/nathanhack/errorcorrectingcodes/cmd/internal/tools/bec"
	"github.com/nathanhack/errorcorrectingcodes/linearblock"
	bec2 "github.com/nathanhack/errorcorrectingcodes/linearblock/messagepassing/bec"
	"github.com/nathanhack/errorcorrectingcodes/linearblock/messagepassing/bec/iterative"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"sync"
	"syscall"
)

var (
	Trials           uint
	ErrorProbability []float64
	Threads          uint
)
var BecRun = func(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		fmt.Println("requires both ECC_JSON_FILE RESULT_JSON")
		return
	}

	//first get the ECC to use
	ecc, err := tools.LoadLinearBlockECC(args[0])
	if err != nil {
		fmt.Println(err)
		return
	}

	//next we see if the RESULT_JSON exists if so we load it and validate we're running it against the right thing
	data, err := tools.LoadResults(args[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	//if data is nil then we create it
	if data == nil {
		data = &tools.SimulationStats{
			TypeInfo: typeInfo(),
			ECCInfo:  tools.Md5Sum(ecc.H),
			Stats:    make(map[float64]benchmarking.Stats),
		}
	}

	//in either case lets validate it
	if data.TypeInfo != typeInfo() {
		fmt.Printf("csv loaded does not match the same type expected %v but found %v\n", typeInfo(), data.TypeInfo)
		return
	}
	if data.ECCInfo != tools.Md5Sum(ecc.H) {
		fmt.Printf("csv laoded does not match the ECC")
		return
	}

	// handle ctrl-C's to kill in a nice way
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		cancel()
	}()

	runSimulation(ctx, data, ecc, args[1])

	err = tools.SaveResults(args[1], data)
	if err != nil {
		fmt.Println(err)
	}
}

func typeInfo() string {
	t := reflect.TypeOf(iterative.Simple{})
	return fmt.Sprintf("BEC:%v/%v", t.PkgPath(), t.Name())

}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func runSimulation(ctx context.Context, data *tools.SimulationStats, ecc *linearblock.LinearBlock, outputFilename string) {
	checkpointMux := sync.Mutex{}
	checkpointCount := 0

	alg := &iterative.Simple{
		H: ecc.H,
	}
	correctionAlg := func(originalCodeword, channelInducedCodeword []bec2.ErasureBit) (fixedChannelInducedCodeword []bec2.ErasureBit) {
		return bec2.Flipping(alg, channelInducedCodeword)
	}

	numberOfThread := int(Threads)
	if numberOfThread == 0 {
		numberOfThread = runtime.NumCPU()
	}

	trialsPerIter := numberOfThread * 10
	bar := pb.StartNew(int(Trials) * len(ErrorProbability))
trialLoops:
	for t := 0; t <= int(Trials); t += trialsPerIter {
		select {
		case <-ctx.Done():
			break trialLoops
		default:
		}

		for _, p := range ErrorProbability {
			checkpoint := func(stats benchmarking.Stats) {
				//we want to save the checkpoint
				checkpointMux.Lock()
				defer checkpointMux.Unlock()

				data.Stats[p] = stats

				if checkpointCount%trialsPerIter == 0 {
					err := tools.SaveResults(outputFilename, data)
					if err != nil {
						fmt.Println(err)
					}
				}
				checkpointCount++
			}
			data.Stats[p] = bec.RunBEC(ctx, ecc, p, min(t, int(Trials)), numberOfThread, correctionAlg, data.Stats[p], checkpoint, false)
			bar.Add(trialsPerIter)
		}
	}
	bar.Finish()
}
