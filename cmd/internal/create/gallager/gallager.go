package gallager

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nathanhack/ecc/linearblock/ldpc/gallager"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var Message uint
var Wc uint
var Wr uint
var Smallest uint
var Iter uint
var Threads uint
var Verbose bool

var GallagerRun = func(cmd *cobra.Command, args []string) {
	//we seed the randomizer so we get something different every time
	rand.Seed(time.Now().Unix())

	if Verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		cancel()
	}()

	g, err := gallager.Search(ctx, int(Message), int(Wc), int(Wr), int(Smallest), int(Iter), int(Threads))
	if err != nil {
		fmt.Println("Unable to create gallager LDPC: ", err)
		return
	}

	if g == nil {
		fmt.Println("Unable to create gallager LDPC try again")
		return
	}

	bs, err := json.Marshal(g)
	if err != nil {
		fmt.Println("Unable to serialize the gallager LDPC: ", err)
		return
	}

	err = ioutil.WriteFile(args[0], bs, 0644)
	if err != nil {
		fmt.Println("unable to write file: ", err)
	}
}
