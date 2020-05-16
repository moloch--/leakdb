package apiclient

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
	Short: "Display LeakDB client version",
	Long:  `Print the version number of leakdb (api client) and exit`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("LeakDB API Client v%s\n", Version)
	},
}
