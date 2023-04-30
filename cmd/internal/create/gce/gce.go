package gce

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

	"github.com/nathanhack/ecc/linearblock"
	"github.com/nathanhack/ecc/linearblock/ldpc/gce"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var MessageSize uint
var CodewordSize uint
var Girth uint
var Iter uint
var Threads uint
var Force bool
var Verbose bool

var GCERun = func(cmd *cobra.Command, args []string) {
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

	checkpoint := func(lb *linearblock.LinearBlock) {
		if lb == nil {
			fmt.Println("Unable to create LDPC try again")
			return
		}

		bs, err := json.Marshal(lb)
		if err != nil {
			fmt.Println("Unable to serialize the LDPC: ", err)
			return
		}

		err = ioutil.WriteFile(args[0], bs, 0644)
		if err != nil {
			fmt.Println("unable to write file: ", err)
		}
	}

	checkNodes := CodewordSize - MessageSize
	g, err := gce.Search(ctx, int(checkNodes), int(CodewordSize), int(Girth), int(Iter), int(Threads), Force, checkpoint)
	if err != nil {
		fmt.Println("Unable to create LDPC: ", err)
		return
	}

	if g == nil {
		fmt.Println("Unable to create LDPC try again")
		return
	}

	bs, err := json.Marshal(g)
	if err != nil {
		fmt.Println("Unable to serialize the LDPC: ", err)
		return
	}

	err = ioutil.WriteFile(args[0], bs, 0644)
	if err != nil {
		fmt.Println("unable to write file: ", err)
	}
}
