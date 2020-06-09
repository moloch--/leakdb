package apiserver

import (
	"fmt"
	"os"

	"github.com/moloch--/leakdb/api"
	"github.com/spf13/cobra"
)

func startServer(cmd *cobra.Command, args []string) {
	server := getServer(cmd, args)
	server.Messages = getNullChannel()
	defer close(server.Messages)
	host, port, err := getHostPort(cmd, args)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.Start(host, port)
}

func startTLSServer(cmd *cobra.Command, args []string) {
	server := getServer(cmd, args)
	server.Messages = getNullChannel()
	defer close(server.Messages)
	cert, key, err := getTLSConfig(cmd, args)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.TLSCertificate = cert
	server.TLSKey = key

	host, port, err := getHostPort(cmd, args)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.StartTLS(host, port)
}

func getHostPort(cmd *cobra.Command, args []string) (string, uint16, error) {
	host, err := cmd.Flags().GetString(hostFlagStr)
	if err != nil {
		return "", 0, fmt.Errorf("Failed to parse --%s flag: %s", hostFlagStr, err)
	}
	port, err := cmd.Flags().GetUint16(portFlagStr)
	if err != nil {
		return "", 0, fmt.Errorf("Failed to parse --%s flag: %s", portFlagStr, err)
	}
	return host, port, nil
}

func getServer(cmd *cobra.Command, args []string) *api.Server {
	jsonFile, err := cmd.Flags().GetString(jsonFlagStr)
	if err != nil {
		fmt.Printf("Failed to parse --%s flag: %s\n", jsonFlagStr, err)
		return nil
	}
	if !fileExists(jsonFile) {
		fmt.Printf("File does not exist %s\n", jsonFile)
		return nil
	}

	emailIndex, err := cmd.Flags().GetString(emailIndexFlagStr)
	if err != nil {
		fmt.Printf("Failed to parse --%s flag: %s\n", emailIndexFlagStr, err)
		return nil
	}
	if emailIndex != "" && !fileExists(emailIndex) {
		fmt.Printf("File does not exist %s", emailIndex)
		return nil
	}

	userIndex, err := cmd.Flags().GetString(userIndexFlagStr)
	if err != nil {
		fmt.Printf("Failed to parse --%s flag: %s\n", userIndexFlagStr, err)
		return nil
	}
	if userIndex != "" && !fileExists(userIndex) {
		fmt.Printf("File does not exist %s", userIndex)
		return nil
	}

	domainIndex, err := cmd.Flags().GetString(domainIndexFlagStr)
	if err != nil {
		fmt.Printf("Failed to parse --%s flag: %s\n", domainIndexFlagStr, err)
		return nil
	}
	if domainIndex != "" && !fileExists(domainIndex) {
		fmt.Printf("File does not exist %s", domainIndex)
		return nil
	}

	return &api.Server{
		JSONFile:    jsonFile,
		EmailIndex:  emailIndex,
		UserIndex:   userIndex,
		DomainIndex: domainIndex,
	}
}

func getTLSConfig(cmd *cobra.Command, args []string) (string, string, error) {
	cert, err := cmd.Flags().GetString(certFlagStr)
	if err != nil {
		return "", "", fmt.Errorf("Failed to parse --%s flag: %s", certFlagStr, err)
	}
	if !fileExists(cert) {
		return "", "", fmt.Errorf("File does not exist %s", cert)
	}
	key, err := cmd.Flags().GetString(keyFlagStr)
	if err != nil {
		return "", "", fmt.Errorf("Failed to parse --%s flag: %s", keyFlagStr, err)
	}
	if !fileExists(key) {
		return "", "", fmt.Errorf("File does not exist %s", key)
	}
	return cert, key, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func getNullChannel() chan string {
	null := make(chan string)
	go func() {
		for range null {
		}
	}()
	return null
}
