package config

import (
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	prefix     = "EDGE_DEVICE"
	logLevel   = "LOG_LEVEL"
	caRoot     = "CA_ROOT"
	certFile   = "CERT"
	privateKey = "PRIVATE_KEY"
	SERVER     = "SERVER"

	gracefulShutdown        = "GRACEFUL_SHUTDOWN"
	defaultGracefulShutdown = 5 * time.Second

	defaultHttpTimeout = 5 * time.Second
)

func ParseConfiguration(confFile string) {
	viper.SetEnvPrefix(prefix)
	viper.AutomaticEnv() // read in environment variables that match

	if len(confFile) == 0 {
		zap.S().Info("no config file specified")
		return
	}

	viper.SetConfigFile(confFile)

	err := viper.ReadInConfig()
	if err != nil {
		zap.S().Errorf("error", err, "config file %v", confFile)
		return
	}

	zap.S().Infof("using config file: %v", viper.ConfigFileUsed())
}

func GetGracefulShutdownDuration() time.Duration {
	if !viper.IsSet(gracefulShutdown) {
		return defaultGracefulShutdown
	}

	return viper.GetDuration(gracefulShutdown)
}

func GetHttpRequestTimeout() time.Duration {
	return defaultHttpTimeout
}
