package cmd

import (
	"github.com/nathanhack/ecc/cmd/internal/create/gallager"
	"github.com/nathanhack/ecc/cmd/internal/create/hamming"
	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"c"},
	Short:   "used to create a new ECC",
	Long:    `create provides the ability to make a new ECC from the list of built-in ECCs and save them so they can be used later by the tools.`,
}

// createlinearblockCmd represents the linearblock command
var createlinearblockCmd = &cobra.Command{
	Use:     "linearblock",
	Aliases: []string{"lb", "l"},
	Short:   "creates linearblock ECCs",
	Long:    `Creates linearblock ECCs.`,
}

// createldpcCmd represents the ldpc command
var createldpcCmd = &cobra.Command{
	Use:     "ldpc",
	Aliases: []string{"l"},
	Short:   "creates LDPC",
	Long:    `Creates linearblock ECCs known as Low Density Parity Check (LDPC)`,
}

// createGallagerCmd represents the gallager command
var createGallagerCmd = &cobra.Command{
	Use:     "gallager OUTPUT_LDPC_JSON",
	Aliases: []string{"g"},
	Short:   "Creates a new Gallager based ECC",
	Long:    `Creates a new Gallager based ECC. Note a small cycle has a negative effect on the effectiveness of the LDPC.`,
	Args:    cobra.ExactArgs(1),
	Run:     gallager.GallagerRun,
}

// createHammingCmd represents the Hamming command
var createHammingCmd = &cobra.Command{
	Use:     "hamming OUTPUT_HAMMING_JSON",
	Aliases: []string{"h", "ham"},
	Short:   "Creates a new Hamming code based ECC",
	Long:    `Creates a new Hamming code based ECC.`,
	Args:    cobra.ExactArgs(1),
	Run:     hamming.HammingRun,
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createlinearblockCmd)
	createlinearblockCmd.AddCommand(createldpcCmd)
	createldpcCmd.AddCommand(createGallagerCmd)

	createGallagerCmd.Flags().UintVarP(&gallager.Message, "message", "m", 1000, "the number of bits in the message")
	createGallagerCmd.Flags().UintVarP(&gallager.Wc, "column", "c", 3, "the column weight (number of ones in the H matrix column) (>=3)")
	createGallagerCmd.Flags().UintVarP(&gallager.Wr, "row", "r", 4, "the row weight (number of ones in the H matrix row) (column < row)")
	createGallagerCmd.Flags().UintVarP(&gallager.Smallest, "smallest", "s", 4, "the smallest allowed cycle: 4, 6, 8...")
	createGallagerCmd.Flags().UintVarP(&gallager.Iter, "iter", "i", 10000, "the number of iterations to try before terminating the search")
	createGallagerCmd.Flags().UintVarP(&gallager.Threads, "threads", "t", 0, "the number of threads to use; note 0 means use the number of cpus")
	createGallagerCmd.Flags().BoolVarP(&gallager.Verbose, "verbose", "v", false, "enable verbose info")

	createlinearblockCmd.AddCommand(createHammingCmd)

	createHammingCmd.Flags().UintVarP(&hamming.ParityBits, "parity", "p", 4, "the parity >=2, sets codeword size (cs) == 2^parity-1 and message size == cs-parity")
	createHammingCmd.Flags().UintVarP(&hamming.Threads, "threads", "t", 0, "the number of threads to use; note 0 means use the number of cpus")

}
