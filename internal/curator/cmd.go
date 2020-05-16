package curator

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "leakdb-curator",
	Short: "Curator datasets for use with LeakDB",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Must specify a command, see --help\n")
	},
}

// Execute - Execute the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
