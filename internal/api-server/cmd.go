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

	tlsFlagStr  = "enable-tls"
	certFlagStr = "cert"
	keyFlagStr  = "key"

	hostFlagStr = "host"
	portFlagStr = "port"

	// Version
	detailsFlagStr = "details"
)

var rootCmd = &cobra.Command{
	Use:   "leakdb-server",
	Short: "LeakDB API server",
	Long:  `Start the LeakDB API server`,
	Run: func(cmd *cobra.Command, args []string) {
		enableTLS, err := cmd.Flags().GetBool(tlsFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag: %s\n", tlsFlagStr, err)
			return
		}
		if enableTLS {
			startTLSServer(cmd, args)
		} else {
			startServer(cmd, args)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().StringP(hostFlagStr, "H", "", "Bind host")
	rootCmd.PersistentFlags().Uint16P(portFlagStr, "p", 8888, "Bind port")

	rootCmd.PersistentFlags().StringP(jsonFlagStr, "j", "", "JSON data set file")
	rootCmd.PersistentFlags().StringP(userIndexFlagStr, "u", "", "User index file")
	rootCmd.PersistentFlags().StringP(emailIndexFlagStr, "e", "", "Email index file")
	rootCmd.PersistentFlags().StringP(domainIndexFlagStr, "d", "", "Domain index file")

	rootCmd.PersistentFlags().BoolP(tlsFlagStr, "s", false, "Enable TLS")
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
