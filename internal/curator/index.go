package curator

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
		key, err := cmd.Flags().GetString(keyFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag: %s\n", keyFlagStr, err)
			return
		}
		if key != "email" && key != "user" && key != "domain" {
			fmt.Printf("Error --%s must be one of: email, user, or domain\n", keyFlagStr)
			return
		}
		cleanup, err := cmd.Flags().GetBool(cleanupFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag: %s\n", cleanupFlagStr, err)
			return
		}
		tempDir, err := cmd.Flags().GetString(tempDirFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag: %s\n", tempDirFlagStr, err)
			return
		}
		if tempDir == "" {
			tempDir, err = ioutil.TempDir("", "leakdb_")
			if err != nil {
				fmt.Printf("Temp error: %s\n", err)
				return
			}
		}
		defer os.RemoveAll(tempDir)

		indexer.Start(target, output, key, workers, cleanup, tempDir, true)
	},
}
