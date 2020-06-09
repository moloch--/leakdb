package apiserver

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
	versionCmd.PersistentFlags().BoolP(detailsFlagStr, "d", false, "detailed version info")
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().StringP(hostFlagStr, "b", "", "Bind host")
	rootCmd.PersistentFlags().Uint16P(portFlagStr, "p", 8888, "Bind port")

	rootCmd.PersistentFlags().StringP(jsonFlagStr, "J", "", "JSON data set file")
	rootCmd.PersistentFlags().StringP(userIndexFlagStr, "U", "", "User index file")
	rootCmd.PersistentFlags().StringP(emailIndexFlagStr, "E", "", "Email index file")
	rootCmd.PersistentFlags().StringP(domainIndexFlagStr, "D", "", "Domain index file")

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
