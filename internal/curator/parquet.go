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
	"time"

	"github.com/moloch--/leakdb/pkg/parqueter"
	"github.com/spf13/cobra"
)

var parquetCmd = &cobra.Command{
	Use:   "parquet",
	Short: "Convert normalized JSON data set into parquet format",
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

		done := make(chan bool)
		parquetConverter, err := parqueter.NewConverter(target, output)
		if err != nil {
			fmt.Printf(Warn+"%s\n", err)
			return
		}
		go func() {
			for {
				select {
				case <-time.After(100 * time.Microsecond):
					fmt.Printf("\r\u001b[2K%d", parquetConverter.LineNumber)
				case <-done:
					fmt.Printf("\r\u001b[2K")
					done <- true
					return
				}
			}
		}()
		err = parquetConverter.Start()
		if err != nil {
			fmt.Printf("\r\u001b[2K"+Warn+"%s\n", err)
		}
		done <- true
		<-done
		fmt.Println("\r\u001b[2KAll done.")
	},
}
