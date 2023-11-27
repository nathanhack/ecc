package chart

import (
	"fmt"
	"os"
	"sort"

	"github.com/nathanhack/ecc/cmd/internal/tools"
	"github.com/spf13/cobra"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

var OutputFile string

var ChartRun = func(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println("requires at least one RESULTS_JSON")
		return
	}

	// loop through all the results files and collect data needed for displaying

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

	//now make the x axis values

	xvalues, xnames := xAxisAndValues(percentagesFloats)

	f, err := os.Create(OutputFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	// create a new bar instance
	bar := charts.NewBar()
	// set some global options like Title/Legend/ToolTip or anything else
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Results",
			Subtitle: "Error Rates",
			Left:     "20%",
		}),
		charts.WithLegendOpts(opts.Legend{Show: true,
			Orient: "vertical",
			// Orient: "horizontal",
			Right: "0",
			Top:   "top",
			Type:  "scroll",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name:      "Error Probability",
			SplitLine: &opts.SplitLine{Show: true},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name:      "Remaining Error",
			SplitLine: &opts.SplitLine{Show: true},
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: true}),
	)

	bar.SetXAxis(xnames)

	// Put data into instance
	for i, s := range stats {
		bar.AddSeries(args[i], series(s, xvalues))
	}

	// Where the magic happens

	bar.Render(f)
}

func xAxisAndValues(percentagesFloats map[float64]bool) ([]float64, []string) {
	nums := make([]float64, 0, len(percentagesFloats))
	strs := make([]string, 0, len(percentagesFloats))
	for k := range percentagesFloats {
		nums = append(nums, k)
	}

	sort.Float64s(nums)

	for _, n := range nums {
		strs = append(strs, fmt.Sprint(n))
	}

	return nums, strs
}

func series(stat *tools.SimulationStats, values []float64) []opts.BarData {
	results := make([]opts.BarData, len(values))
	null := opts.BarData{Value: nil}
	for i, v := range values {

		x, has := stat.Stats[v]
		if !has {
			results[i] = null
			continue
		}

		results[i] = opts.BarData{
			Value: x.ChannelCodewordError.Mean,
		}
	}
	return results
}
