/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"encoding/json"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	config "github.com/tupyy/device-worker-ng/configuration"
	"github.com/tupyy/device-worker-ng/internal/certificate"
	"github.com/tupyy/device-worker-ng/internal/client"
	"github.com/tupyy/device-worker-ng/internal/configuration"
	"github.com/tupyy/device-worker-ng/internal/edge"
	"go.uber.org/zap"
)

var (
	configFile string
	caRoot     string
	certFile   string
	privateKey string
	server     string
	namespace  string
	logLevel   string
)

var rootCmd = &cobra.Command{
	Use:   "device-worker-ng",
	Short: "Device worker",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return config.InitConfiguration(cmd, configFile)
	},
	Run: func(cmd *cobra.Command, args []string) {
		logger := setupLogger()
		defer logger.Sync()

		undo := zap.ReplaceGlobals(logger)
		defer undo()

		certManager, err := initCertificateManager(caRoot, certFile, privateKey)
		if err != nil {
			panic(err)
		}

		tlsConfig, err := certManager.TLSConfig()
		if err != nil {
			panic(err)
		}

		// httpClient is a wrapper around yggdrasil http client.
		httpClient, err := client.New(config.GetServerAddress(), tlsConfig)
		if err != nil {
			panic(err)
		}

		confManager := configuration.New()

		controller := edge.New(httpClient, confManager, certManager)

		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, os.Kill)

		<-done

		controller.Shutdown()

	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&configFile, "config", "c", "configuration file")
	rootCmd.Flags().StringVar(&caRoot, "ca-root", "", "ca certificate")
	rootCmd.Flags().StringVar(&certFile, "cert", "", "client certificate")
	rootCmd.Flags().StringVar(&privateKey, "key", "", "private key")
	rootCmd.Flags().StringVar(&server, "server", "", "server address")
	rootCmd.Flags().StringVar(&namespace, "namespace", "default", "target namespace")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "log level")
}

func setupLogger() *zap.Logger {
	rawJSON := []byte(`{
	  "level": "info",
	  "encoding": "json",
	  "outputPaths": ["stdout"],
	  "errorOutputPaths": ["stderr"],
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
	    "levelEncoder": "lowercase"
	  }
	}`)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}

	atomicLogLevel, err := zap.ParseAtomicLevel(logLevel)
	if err == nil {
		cfg.Level = atomicLogLevel
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	return logger
}

func initCertificateManager(caroot, certFile, keyFile string) (*certificate.Manager, error) {
	// read certificates
	caRoot, err := os.ReadFile(caroot)
	if err != nil {
		return nil, err
	}

	cert, err := os.ReadFile(certFile)
	if err != nil {
		return nil, err
	}

	privateKey, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	certManager, err := certificate.New([][]byte{caRoot}, cert, privateKey)
	if err != nil {
		return nil, err
	}

	return certManager, nil
}
