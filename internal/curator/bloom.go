package curator

/*
	---------------------------------------------------------------------
	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <http://www.gnu.org/licenses/>.
	----------------------------------------------------------------------
*/

import (
	"fmt"
	"os"

	"github.com/moloch--/leakdb/pkg/bloomer"
	"github.com/spf13/cobra"
)

var bloomCmd = &cobra.Command{
	Use:   "bloom",
	Short: "Bloom filter",
	Long:  `Apply bloom filter to remove duplicates`,
	Run: func(cmd *cobra.Command, args []string) {
		target, err := cmd.Flags().GetString(jsonFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", jsonFlagStr, err)
			return
		}
		output, err := cmd.Flags().GetString(outputFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", outputFlagStr, err)
			return
		}
		workers, err := cmd.Flags().GetUint(workersFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", workersFlagStr, err)
			return
		}
		filterSize, err := cmd.Flags().GetUint(filterSizeFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", filterSizeFlagStr, err)
			return
		}
		filterHashes, err := cmd.Flags().GetUint(filterHashesFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", filterHashesFlagStr, err)
			return
		}
		filterLoad, err := cmd.Flags().GetString(filterLoadFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", filterLoadFlagStr, err)
			return
		}
		filterSave, err := cmd.Flags().GetString(filterSaveFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", filterSaveFlagStr, err)
			return
		}

		if _, err = os.Stat(target); os.IsNotExist(err) {
			fmt.Printf(Warn+"Target error: %s", err)
			return
		}

		fmt.Printf(Info + "Bloom Filter:\n")
		fmt.Printf("\tSize = %dGb (%d bytes)\n", filterSize, (filterSize * gb))
		fmt.Printf("\tHashes = %d\n", filterHashes)
		fmt.Println()
		fmt.Printf(Info+"Target: %v\n", target)
		fmt.Printf(Info+"Output: %s\n", output)

		bloom, err := bloomer.GetBloomer(target, output, filterSave, filterLoad, workers, filterSize, filterHashes)
		if err != nil {
			fmt.Printf(Warn+"Bloom error %s", err)
		}
		done := make(chan bool)
		go bloomProgress(bloom, done)
		err = bloom.Start()
		done <- true
		<-done
		if err != nil {
			fmt.Printf(Warn+"Bloom error %s", err)
		}
	},
}
