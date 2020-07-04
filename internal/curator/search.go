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
	"text/tabwriter"

	"github.com/moloch--/leakdb/pkg/searcher"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search an index for an entry",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		target, err := cmd.Flags().GetString(jsonFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", jsonFlagStr, err)
			return
		}
		index, err := cmd.Flags().GetString(indexFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", indexFlagStr, err)
			return
		}
		value, err := cmd.Flags().GetString(valueFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", valueFlagStr, err)
			return
		}
		if value == "" {
			fmt.Printf(Warn+"Must specify --%s\n", valueFlagStr)
			return
		}

		credentials, err := searcher.Start(value, target, index)
		if err != nil {
			fmt.Printf(Warn+"%s\n", err)
			return
		}
		fmt.Printf("Found %d results ...\n", len(credentials))
		if 0 < len(credentials) {
			// displayCredentials(credentials)
			for _, cred := range credentials {
				fmt.Printf("%v\n", cred)
			}
		}
	},
}

func displayCredentials(credentials []*searcher.Credential) {
	table := new(tabwriter.Writer)
	table.Init(os.Stdout, 1, 4, 2, ' ', 0)
	fmt.Fprintf(table, "Email\tUser\tDomain\tPassword\n")
	fmt.Fprintf(table, "=====\t====\t======\t========\n")
	for _, cred := range credentials {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", cred.Email, cred.User, cred.Domain, cred.Password)
	}
	table.Flush()
}
