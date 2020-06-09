package apiserver

import (
	"fmt"
	"strconv"
	"time"

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
	Long:  `Print the version number of leakdb-server and exit`,
	Run: func(cmd *cobra.Command, args []string) {
		details, _ := cmd.Flags().GetBool(detailsFlagStr)
		if details {
			timeStamp, _ := strconv.Atoi(CompiledAt)
			compiledAt := time.Unix(int64(timeStamp), 0)
			fmt.Printf("LeakDB API Server v%s - Compiled %s - %s\n", Version, compiledAt, GitCommit)
		} else {
			fmt.Printf("LeakDB API Server v%s\n", Version)
		}
	},
}
