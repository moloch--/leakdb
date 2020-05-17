package curator

import (
	"bufio"
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
		verbose, err := cmd.Flags().GetBool(verboseFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", verboseFlagStr, err)
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

		messages := make(chan string)
		go func() {
			stdout := bufio.NewWriter(os.Stdout)
			for message := range messages {
				if verbose {
					stdout.Write([]byte(message))
					stdout.Flush()
				}
			}
		}()
		defer close(messages)

		credentials, err := searcher.Start(messages, value, target, index)
		if err != nil {
			fmt.Printf(Warn+"%s\n", err)
			return
		}

		displayCredentials(credentials)
	},
}

func displayCredentials(credentials []searcher.Credential) {
	table := new(tabwriter.Writer)
	table.Init(os.Stdout, 1, 4, 2, ' ', 0)
	fmt.Fprintln(table, "Email\tUser\tDomain\tPassword")
	fmt.Fprintln(table, "=====\t====\t======\t========")
	for _, cred := range credentials {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", cred.Email, cred.User, cred.Domain, cred.Password)
	}
	fmt.Fprintln(table)
	table.Flush()
}
