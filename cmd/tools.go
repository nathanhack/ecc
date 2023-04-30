package cmd

import (
	"github.com/nathanhack/ecc/cmd/internal/tools/bec/simple"
	"github.com/nathanhack/ecc/cmd/internal/tools/bsc/dwbf"
	"github.com/nathanhack/ecc/cmd/internal/tools/bsc/gallager"
	"github.com/nathanhack/ecc/cmd/internal/tools/csv"

	"github.com/spf13/cobra"
)

// toolsCmd represents the tools command
var toolsCmd = &cobra.Command{
	Use:     "tools",
	Aliases: []string{"t"},
	Short:   "Tools for ECCs",
	Long:    `Tools for ECCs`,
}

// toolsChansimCmd represents the chansim command
var toolsChansimCmd = &cobra.Command{
	Use:     "chansim",
	Aliases: []string{"cs", "c"},
	Short:   "Channel simulators",
	Long:    `Channel simulators for linearblock ECCs`,
}

// toolsLinearblockCmd represents the linearblock command
var toolsLinearblockCmd = &cobra.Command{
	Use:     "linearblock",
	Aliases: []string{"lb", "l"},
	Short:   "Linearblock channel simulators",
	Long:    `Channel simulators for linearblock ECCs`,
}

// toolsHarddecisionCmd represents the harddecision command
var toolsHarddecisionCmd = &cobra.Command{
	Use:     "harddecision",
	Aliases: []string{"hard", "h"},
	Short:   "Using hard decisions",
	Long:    `Channel simulators for linearblock ECCs using hard decisions`,
}

// toolsBecCmd represents the bec command
var toolsBecCmd = &cobra.Command{
	Use:   "bec ECC_JSON_FILE RESULT_JSON",
	Short: "An erasure channel simulator",
	Long:  `A simple erasure channel simulator for linearblock ECCs`,
	Run:   simple.BecRun,
}

// toolsBscCmd represents the bsc command
var toolsBscCmd = &cobra.Command{
	Use:   "bsc",
	Short: "A binary symmetric channel simulator",
	Long:  `A binary symmetric channel simulator for linearblock ECCs`,
}

// toolsDwbfCmd represents the dwbf command
var toolsDwbfCmd = &cobra.Command{
	Use:     "dwbf ECC_JSON_FILE RESULT_JSON",
	Aliases: []string{"d"},
	Short:   "A linearblock BSC simulator with dwbf based bit flipping algorithm",
	Long:    `A linearblock BSC simulator with dwbf based bit flipping algorithm`,
	Run:     dwbf.DwbfRun,
}

// toolsGallagerCmd represents the gallager command
var toolsGallagerCmd = &cobra.Command{
	Use:     "gallager ECC_JSON_FILE RESULT_JSON",
	Aliases: []string{"g"},
	Short:   "A linearblock BSC simulator with gallager based bit flipping algorithm",
	Long:    `A linearblock BSC simulator with gallager based bit flipping algorithm`,
	Run:     gallager.GallagerRun,
}

// toolsResultsCmd represents the csv command
var toolsResultsCmd = &cobra.Command{
	Use:     "results",
	Aliases: []string{"r"},
	Short:   "A tool to organize results for graphing and comparison",
	Long:    `A tool to organize results for graphing and comparison`,
}

// toolsCSVCmd represents the csv command
var toolsCSVCmd = &cobra.Command{
	Use:     "csv RESULTS_JSON [RESULTS_JSON] ...",
	Aliases: []string{"c"},
	Short:   "Export to a CSV file",
	Long:    `Export to a CSV file`,
	Run:     csv.CSVRun,
}

func init() {
	rootCmd.AddCommand(toolsCmd)
	toolsCmd.AddCommand(toolsChansimCmd)
	toolsCmd.AddCommand(toolsResultsCmd)

	toolsChansimCmd.AddCommand(toolsLinearblockCmd)
	toolsLinearblockCmd.AddCommand(toolsHarddecisionCmd)

	toolsHarddecisionCmd.AddCommand(toolsBecCmd)
	toolsBecCmd.Flags().UintVarP(&simple.Trials, "trials", "t", 1_000_000, "the number of trials per step")
	toolsBecCmd.Flags().Float64SliceVarP(&simple.ErrorProbability, "probability", "p", []float64{0.01, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.99}, "probability of erasure [0, 1)")
	toolsBecCmd.Flags().UintVar(&simple.Threads, "threads", 0, "number of threads to use (0 means to use the # of threads equal to the # of CPUs)")

	toolsHarddecisionCmd.AddCommand(toolsBscCmd)

	toolsBscCmd.AddCommand(toolsDwbfCmd)
	toolsDwbfCmd.Flags().UintVarP(&dwbf.Trials, "trials", "t", 1_000_000, "the number of trials per step")
	toolsDwbfCmd.Flags().Float64SliceVarP(&dwbf.ErrorProbability, "probability", "p", []float64{0.01, 0.05, 0.10, 0.15, 0.20, 0.25, 0.30, 0.35, 0.40, 0.45, 0.50}, "probability of crossover errors to test [0, 0.5]")
	toolsDwbfCmd.Flags().UintVar(&dwbf.Threads, "threads", 0, "number of threads to use (0 means to use the # of threads equal to the # of CPUs)")
	toolsDwbfCmd.Flags().UintVarP(&dwbf.MaxIter, "iters", "i", 20, "max number of iterations the bitflip algorithm is allowed")
	toolsDwbfCmd.Flags().Float64VarP(&dwbf.Alpha, "alpha", "a", .5, "hyperparameter 0<α<1")
	toolsDwbfCmd.Flags().Float64VarP(&dwbf.EtaThreshold, "eta", "e", 0.0, "hyperparameter η threshold: no requirement but frequently 0.0 is a good value")

	toolsBscCmd.AddCommand(toolsGallagerCmd)

	toolsGallagerCmd.Flags().UintVarP(&gallager.Trials, "trials", "t", 1_000_000, "the number of trials per step")
	toolsGallagerCmd.Flags().Float64SliceVarP(&gallager.ErrorProbability, "probability", "p", []float64{0.01, 0.05, 0.10, 0.15, 0.20, 0.25, 0.30, 0.35, 0.40, 0.45, 0.50}, "probability of crossover errors to test [0, 0.5]")
	toolsGallagerCmd.Flags().UintVar(&gallager.Threads, "threads", 0, "number of threads to use (0 means to use the # of threads equal to the # of CPUs)")
	toolsGallagerCmd.Flags().UintVarP(&gallager.MaxIter, "iters", "i", 20, "max number of iterations the bitflip algorithm is allowed")

	toolsResultsCmd.AddCommand(toolsCSVCmd)
	toolsCSVCmd.Flags().StringVarP(&csv.OutputFile, "output", "o", "results.csv", "filename of the combined csv")
	toolsCSVCmd.Flags().BoolVarP(&csv.MessageError, "message", "m", false, "outputs the MessageError instead of CodewordError or ParityError")
	toolsCSVCmd.Flags().BoolVarP(&csv.ParityError, "parity", "p", false, "outputs the ParityError instead of CodewordError or MessageError")
}
