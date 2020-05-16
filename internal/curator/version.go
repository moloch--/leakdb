package curator

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version - The semantic version of the program
	Version string

	// CompiledAt - Unix timestamp of compiled binary
	CompiledAt string

	// GitCommit - Most recent commit from which the binary was compiled
	GitCommit string

	// GitDirty - Set if the compiled from uncommitted code
	GitDirty string
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long:  `Print the version number of leakdb-curator and exit`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("LeakDB Curator v%s\n", Version)
	},
}
