/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	config "github.com/tupyy/device-worker-ng/configuration"
	"github.com/tupyy/device-worker-ng/internal/certificate"
	httpClient "github.com/tupyy/device-worker-ng/internal/client/http"
	"github.com/tupyy/device-worker-ng/internal/configuration"
	"github.com/tupyy/device-worker-ng/internal/edge"
	"github.com/tupyy/device-worker-ng/internal/executor"
	"github.com/tupyy/device-worker-ng/internal/scheduler"
	"github.com/tupyy/device-worker-ng/internal/state"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

		// httpClient is a wrapper around http client which implements yggdrasil API.
		httpClient, err := httpClient.New(config.GetServerAddress(), certManager)
		if err != nil {
			panic(err)
		}

		confManager := configuration.New()
		executor, err := executor.New()
		if err != nil {
			panic(err)
		}

		controller := edge.New(httpClient, confManager, certManager)
		stateManager := state.New(confManager.ProfileCh)
		scheduler := scheduler.New(executor)

		ctx, cancel := context.WithCancel(context.Background())
		scheduler.Start(ctx, confManager.TaskCh)

		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, os.Kill)

		<-done

		cancel()
		controller.Shutdown()
		stateManager.Shutdown()

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
	loggerCfg := &zap.Config{
		Level:    zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "severity",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: zapcore.MillisDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	atomicLogLevel, err := zap.ParseAtomicLevel(logLevel)
	if err == nil {
		loggerCfg.Level = atomicLogLevel
	}

	plain, err := loggerCfg.Build(zap.AddStacktrace(zap.DPanicLevel))
	if err != nil {
		panic(err)
	}

	return plain
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
