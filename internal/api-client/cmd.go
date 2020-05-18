package apiclient

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
	"time"

	"github.com/moloch--/leakdb/api"
	"github.com/moloch--/leakdb/pkg/leakdb"
	"github.com/spf13/cobra"
)

var (
	// URL - Service URL
	URL = getEnvVar("LEAKDB_URL", "")

	// APIToken - Auth token
	APIToken = getEnvVar("LEAKDB_API_TOKEN", "")
)

// OutputConfig - Configure the client output of a query
type OutputConfig struct {
	PasswordOnly bool
	EmailOnly    bool
	FilterBlank  bool
	FilterHashes bool
}

var rootCmd = &cobra.Command{
	Use:   "leakdb",
	Short: "LeakDB cli client",
	Long:  `Query LeakDB for leaked credentials based on email, user, or domain.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Must specify a query, see --help\n")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Pagination
	rootCmd.PersistentFlags().IntP("page", "p", 0, "Page number")

	// Output options
	rootCmd.PersistentFlags().StringP("save", "s", "", "Save results to file")
	rootCmd.PersistentFlags().BoolP("email-only", "e", false, "Output emails only")
	rootCmd.PersistentFlags().BoolP("password-only", "w", false, "Output passwords only")
	rootCmd.PersistentFlags().BoolP("no-empty", "t", false, "Filter results that appear to contain an empty password")
	rootCmd.PersistentFlags().BoolP("no-hashes", "n", false, "Filter results that appear to contain a password hash")

	// Proxy options
	rootCmd.PersistentFlags().BoolP("skip-tls-validation", "V", false, "Skip TLS certificate validation")
	rootCmd.PersistentFlags().StringP("proxy", "H", "", "Specify HTTP(S) proxy URL (e.g. http://localhost:8080)")
	rootCmd.PersistentFlags().IntP("timeout", "T", 30, "HTTPS request/connection timeout")

	rootCmd.AddCommand(emailCmd)
	rootCmd.AddCommand(domainCmd)
	rootCmd.AddCommand(userCmd)
}

func parsePaginationFlags(cmd *cobra.Command) (int, error) {
	page, err := cmd.Flags().GetInt("page")
	if err != nil {
		fmt.Printf("Failed to parse --page flag: %s\n", err)
		return 0, err
	}
	return page, nil
}

func parseHTTPFlags(cmd *cobra.Command) (leakdb.ClientHTTPConfig, error) {
	skipTLSValidation, err := cmd.Flags().GetBool("skip-tls-validation")
	if err != nil {
		fmt.Printf("Failed to parse --skip-tls-validation flag: %s\n", err)
		return leakdb.ClientHTTPConfig{}, err
	}
	proxyURL, err := cmd.Flags().GetString("proxy")
	if err != nil {
		fmt.Printf("Failed to parse --proxy flag: %s\n", err)
		return leakdb.ClientHTTPConfig{}, err
	}
	timeout, err := cmd.Flags().GetInt("timeout")
	if err != nil {
		fmt.Printf("Failed to parse --timeout flag: %s\n", err)
		return leakdb.ClientHTTPConfig{}, err
	}
	return leakdb.ClientHTTPConfig{
		ProxyURL:          proxyURL,
		SkipTLSValidation: skipTLSValidation,
		Timeout:           time.Duration(timeout) * time.Second,
	}, nil
}

func parseOutputFlags(cmd *cobra.Command) (string, *OutputConfig, error) {

	config := &OutputConfig{}

	save, err := cmd.Flags().GetString("save")
	if err != nil {
		fmt.Printf("Failed to parse --save flag: %s\n", err)
		return "", config, err
	}

	config.EmailOnly, err = cmd.Flags().GetBool("email-only")
	if err != nil {
		fmt.Printf("Failed to parse --email-only flag: %s\n", err)
		return save, config, err
	}
	config.PasswordOnly, err = cmd.Flags().GetBool("password-only")
	if err != nil {
		fmt.Printf("Failed to parse --password-only flag: %s\n", err)
		return save, config, err
	}
	config.FilterBlank, err = cmd.Flags().GetBool("no-empty")
	if err != nil {
		fmt.Printf("Failed to parse --no-empty flag: %s\n", err)
		return save, config, err
	}
	config.FilterHashes, err = cmd.Flags().GetBool("no-hashes")
	if err != nil {
		fmt.Printf("Failed to parse --no-hashes flag: %s\n", err)
		return save, config, err
	}

	return save, config, nil
}

func genericQueryCommand(cmd *cobra.Command, querySet *api.QuerySet) {
	page, err := parsePaginationFlags(cmd)
	if err != nil {
		return
	}
	querySet.Page = page

	httpConfig, err := parseHTTPFlags(cmd)
	if err != nil {
		return
	}

	save, outputConfig, err := parseOutputFlags(cmd)
	if err != nil {
		return
	}

	client, err := leakdb.NewClient(URL, APIToken, httpConfig)
	if err != nil {
		fmt.Printf("[error] HTTP client failure: %v\n", err)
		return
	}
	results, err := client.Query(querySet)
	if err != nil {
		fmt.Printf("[error] Failed to parse response: %v\n", err)
		return
	}

	display(outputConfig, results)
	if save != "" {
		saveToFile(save, outputConfig, results)
	}
}

func saveToFile(save string, conf *OutputConfig, results *api.ResultSet) {
	saveFile, err := os.OpenFile(save, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		fmt.Printf("Failed to save results: %s\n", err)
		return
	}
	defer saveFile.Close()

	fmt.Printf("Saving results to %s ... ", save)
	for _, cred := range results.Results {
		if conf.FilterBlank && cred.IsBlank() {
			continue
		}
		if conf.FilterHashes && cred.IsHash() {
			continue
		}
		data := []byte{}
		if !conf.PasswordOnly {
			data = append(data, []byte(cred.Email)...)
		}
		if !conf.PasswordOnly && !conf.EmailOnly {
			data = append(data, ':')
		}
		if !conf.EmailOnly {
			data = append(data, []byte(cred.Password)...)
		}
		data = append(data, '\n')
		saveFile.Write(data)
	}

	fmt.Printf("done!\n")
}

func display(conf *OutputConfig, results *api.ResultSet) {
	if results.Count == 0 {
		fmt.Println("No results for query")
		return
	}
	if conf.EmailOnly {
		for _, cred := range results.Results {
			fmt.Printf("%s\n", cred.Email)
		}
	} else if conf.PasswordOnly {
		for _, cred := range results.Results {
			fmt.Printf("%s\n", cred.Password)
		}
	} else {
		stdout := tabwriter.NewWriter(os.Stdout, 1, 0, 1, ' ', 0)
		row := 0
		for _, cred := range results.Results {
			if conf.FilterBlank && cred.IsBlank() {
				continue
			}
			if conf.FilterHashes && cred.IsHash() {
				continue
			}
			row++
			fmt.Fprintln(stdout, fmt.Sprintf("%d\t%s\t%s", row, cred.Email, cred.Password))
		}
		stdout.Flush()
		fmt.Println()
		fmt.Printf("Displaying page %d of %d (%d total results)\n",
			results.Page+1, results.Pages+1, results.Count)
	}
}

func getEnvVar(name, defaultValue string) string {
	value := os.Getenv(name)
	if value == "" {
		value = defaultValue
	}
	return value
}

// Execute - Execute the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
