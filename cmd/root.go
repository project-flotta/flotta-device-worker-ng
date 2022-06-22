/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	configFile string
	caRoot     string
	certFile   string
	privateKey string
	server     string
)

var rootCmd = &cobra.Command{
	Use:   "device-worker-ng",
	Short: "Device worker",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StrinvVar(&configFile, "config", "c", false, "configuration file")
	rootCmd.Flags().StringVar(&caRoot, "ca-root", "", false, "ca root file")
	rootCmd.Flags().StringVar(&certFile, "cert-file", "", false, "cert file")
	rootCmd.Flags().StringVar(&privateKey, "key", "", false, "private key")
	rootCmd.Flags().StringVar(&server, "server", "", false, "server address")
}
