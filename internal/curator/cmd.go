package curator

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

const (
	targetFlagStr = "target"
	outputFlagStr = "output"

	workersFlagStr = "workers"

	// Filter flags
	filterSizeFlagStr   = "filter-size"
	filterHashesFlagStr = "filter-hashes"
	filterLoadFlagStr   = "filter-load"
	filterSaveFlagStr   = "filter-save"

	kb = 1024
	mb = kb * 1024
	gb = mb * 1024
)

var rootCmd = &cobra.Command{
	Use:   "leakdb-curator",
	Short: "Curate data sets for use with LeakDB",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Must specify a command, see --help\n")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Bloom
	bloomCmd.Flags().StringP(targetFlagStr, "t", "", "target input file(s)")
	bloomCmd.Flags().StringP(outputFlagStr, "o", "", "output file")
	bloomCmd.Flags().UintP(workersFlagStr, "w", uint(runtime.NumCPU()), "number of worker threads")
	bloomCmd.Flags().UintP(filterSizeFlagStr, "s", 8, "bloom filter size in GBs")
	bloomCmd.Flags().UintP(filterHashesFlagStr, "f", 4, "number of bloom filter hash functions")
	bloomCmd.Flags().StringP(filterLoadFlagStr, "L", "", "load existing bloom filter from saved file")
	bloomCmd.Flags().StringP(filterSaveFlagStr, "S", "", "save bloom filter to file when complete")

	rootCmd.AddCommand(bloomCmd)
}

// Execute - Execute the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
