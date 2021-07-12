package csv

import (
	"encoding/csv"
	"fmt"
	"github.com/nathanhack/errorcorrectingcodes/cmd/internal/tools"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var OutputFile string
var MessageError bool
var ParityError bool

var CSVRun = func(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println("requires at least one RESULTS_JSON")
		return
	}

	stats := make([]*tools.SimulationStats, len(args))
	var err error
	percentagesFloats := make(map[float64]bool)
	for i, resultFile := range args {
		stats[i], err = tools.LoadResults(resultFile)
		if err != nil {
			fmt.Println(err)
			return
		}
		for p := range stats[i].Stats {
			percentagesFloats[p] = true
		}
	}

	f, err := os.Create(OutputFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()

	//first write headers
	percentagesList := make([]float64, 0, len(percentagesFloats))
	for p := range percentagesFloats {
		percentagesList = append(percentagesList, p)
	}
	sort.Float64s(percentagesList)

	header := []string{"Results File"}

	for _, p := range percentagesList {
		header = append(header, fmt.Sprintf("%v", p))
	}

	err = w.Write(header)
	if err != nil {
		fmt.Println(err)
		return
	}

	for i, s := range stats {
		record := make([]string, len(header))
		record[0] = strings.TrimSuffix(args[i], filepath.Ext(args[i]))

		for i, p := range percentagesList {
			v, has := s.Stats[p]
			if has {
				switch {
				case MessageError:
					record[i+1] = fmt.Sprintf("%v", v.ChannelMessageError.Mean)
				case ParityError:
					record[i+1] = fmt.Sprintf("%v", v.ChannelParityError.Mean)
				default:
					record[i+1] = fmt.Sprintf("%v", v.ChannelCodewordError.Mean)
				}
			}
		}

		err = w.Write(record)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}
