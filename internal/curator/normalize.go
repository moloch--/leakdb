package curator

import (
	"fmt"
	"time"

	"github.com/moloch--/leakdb/pkg/normalizer"
	"github.com/spf13/cobra"
)

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

var normalizeCmd = &cobra.Command{
	Use:   "normalize",
	Short: "Normalize data sets",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		target, err := cmd.Flags().GetString(targetFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", targetFlagStr, err)
			return
		}
		output, err := cmd.Flags().GetString(outputFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", outputFlagStr, err)
			return
		}
		skipPrefix, err := cmd.Flags().GetString(skipPrefixFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", skipPrefixFlagStr, err)
			return
		}
		skipSuffix, err := cmd.Flags().GetString(skipSuffixFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", skipSuffixFlagStr, err)
			return
		}
		recursive, err := cmd.Flags().GetBool(recursiveFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", recursiveFlagStr, err)
			return
		}

		// Get format
		targetFormat, err := cmd.Flags().GetString(formatFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", formatFlagStr, err)
			return
		}
		format, supported := normalizer.Formats[targetFormat]
		if !supported {
			fmt.Printf(Warn+"'%s' is not a supported format, see --help\n", targetFormat)
			return
		}

		normalize, err := normalizer.GetNormalizer(format, target, output, recursive, skipPrefix, skipSuffix)
		if err != nil {
			fmt.Printf(Warn+"%s\n", err)
			return
		}

		done := make(chan bool)
		go normalizeProgress(normalize, done)
		start := time.Now()
		normalize.Start()
		done <- true
		<-done
		fmt.Printf("\r\u001b[2KCompleted in %s\n", time.Now().Sub(start))

	},
}

func normalizeProgress(normalize *normalizer.Normalize, done chan bool) {
	for {
		select {
		case <-time.After(time.Second):
			target, line := normalize.GetStatus()
			fmt.Printf("\r\u001b[2K Normalizing %s (line %d) ...", target, line)
		case <-done:
			fmt.Printf("\r\u001b[2K")
			done <- true
			return
		}
	}
}
