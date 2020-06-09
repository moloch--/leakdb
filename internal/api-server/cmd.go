package apiserver

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	jsonFlagStr        = "json"
	userIndexFlagStr   = "index-user"
	emailIndexFlagStr  = "index-email"
	domainIndexFlagStr = "index-domain"

	certFlagStr = "tls-cert"
	keyFlagStr  = "tls-key"

	// Version
	detailsFlagStr = "details"
)

var rootCmd = &cobra.Command{
	Use:   "leakdb-server",
	Short: "LeakDB api server",
	Long:  ``,
	Run:   startServer,
}

func init() {
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().StringP(jsonFlagStr, "j", "", "JSON data set file")
	rootCmd.PersistentFlags().StringP(userIndexFlagStr, "u", "", "User index file")
	rootCmd.PersistentFlags().StringP(emailIndexFlagStr, "e", "", "Email index file")
	rootCmd.PersistentFlags().StringP(domainIndexFlagStr, "d", "", "Domain index file")

	rootCmd.PersistentFlags().StringP(certFlagStr, "c", "", "TLS certificate")
	rootCmd.PersistentFlags().StringP(keyFlagStr, "k", "", "TLS private key")
}

// Execute - Execute the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
