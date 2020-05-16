package curator

import (
	"fmt"

	"github.com/moloch--/leakdb/pkg/bloomer"
	"github.com/spf13/cobra"
)

var bloomCmd = &cobra.Command{
	Use:   "bloom",
	Short: "Bloom filter",
	Long:  `Apply bloom filter to remove duplicates`,
	Run: func(cmd *cobra.Command, args []string) {
		target, err := cmd.Flags().GetString(targetFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag: %s\n", targetFlagStr, err)
			return
		}
		output, err := cmd.Flags().GetString(outputFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag: %s\n", outputFlagStr, err)
			return
		}
		workers, err := cmd.Flags().GetUint(workersFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag: %s\n", workersFlagStr, err)
			return
		}
		filterSize, err := cmd.Flags().GetUint(filterSizeFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag: %s\n", filterSizeFlagStr, err)
			return
		}
		filterHashes, err := cmd.Flags().GetUint(filterHashesFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag: %s\n", filterHashesFlagStr, err)
			return
		}
		filterLoad, err := cmd.Flags().GetString(filterLoadFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag: %s\n", filterLoadFlagStr, err)
			return
		}
		filterSave, err := cmd.Flags().GetString(filterSaveFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag: %s\n", filterSaveFlagStr, err)
			return
		}

		targets, err := bloomer.GetTargets(target)
		if err != nil {
			fmt.Printf("Target error: %s", err)
			return
		}
		fmt.Printf("Bloom Filter:\n\tSize = %dGb (%d bytes)\n\tHashes = %d\n", filterSize, (filterSize * gb), filterHashes)
		fmt.Printf("Target: %v\n", targets)
		fmt.Printf("Output: %s\n", output)

		bloomer.Start(targets, output, filterSave, filterLoad, workers, filterSize, filterHashes)
	},
}
