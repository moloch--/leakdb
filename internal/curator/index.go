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
	"io/ioutil"
	"os"

	"github.com/moloch--/leakdb/pkg/indexer"
	"github.com/spf13/cobra"
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index a target file",
	Long:  `Compute index of a JSON file`,
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
		key, err := cmd.Flags().GetString(keyFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", keyFlagStr, err)
			return
		}
		if key != "email" && key != "user" && key != "domain" {
			fmt.Printf(Warn+"Error --%s must be one of: email, user, or domain\n", keyFlagStr)
			return
		}
		noCleanup, err := cmd.Flags().GetBool(noCleanupFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", noCleanupFlagStr, err)
			return
		}
		tempDir, err := cmd.Flags().GetString(tempDirFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", tempDirFlagStr, err)
			return
		}
		if tempDir == "" {
			tempDir, err = ioutil.TempDir("", "leakdb_")
			if err != nil {
				fmt.Printf(Warn+"Temp error: %s\n", err)
				return
			}
		}
		if !noCleanup {
			defer os.RemoveAll(tempDir)
		}

		index, err := indexer.GetIndexer(target, output, key, workers, tempDir, noCleanup)
		if err != nil {
			fmt.Printf(Warn+"%s\n", err)
		}
		index.Start()
	},
}
