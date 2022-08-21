package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	prefix     = "EDGE_DEVICE"
	logLevel   = "LOG_LEVEL"
	caRoot     = "CA_ROOT"
	certFile   = "CERT"
	privateKey = "KEY"
	server     = "SERVER"
	namespace  = "NAMESPACE"
	deviceID   = "DEVICE_ID"

	gracefulShutdown        = "GRACEFUL_SHUTDOWN"
	defaultGracefulShutdown = 5 * time.Second
	defaultNamespace        = "default"

	defaultHttpTimeout = 5 * time.Second
)

type RetryConfig struct {
	InitialInterval time.Duration
	Multiplier      float64
	MaxInterval     time.Duration
	MaxElapsedTime  time.Duration
}

var v *viper.Viper

func InitConfiguration(cmd *cobra.Command, configFile string) error {
	v = viper.New()

	v.SetEnvPrefix(prefix)
	v.AutomaticEnv() // read in environment variables that match

	if len(configFile) > 0 {
		zap.S().Infof("using config file: %v", viper.ConfigFileUsed())
		v.SetConfigFile(configFile)

		err := v.ReadInConfig()
		if err != nil {
			zap.S().Errorw("error", err, "config file", configFile)
			return fmt.Errorf("fail to read config file")
		}
	}

	// Bind the current command's flags to viper
	bindFlags(cmd, v)

	return nil
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// replace - with _ to match yaml format
		flagName := f.Name
		if strings.Contains(f.Name, "-") {
			// Environment variables can't have dashes in them, so bind them to their equivalent
			// keys with underscores.
			envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
			v.BindEnv(f.Name, fmt.Sprintf("%s_%s", prefix, envVarSuffix))
			flagName = strings.ReplaceAll(f.Name, "-", "_")
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		// and the other way around.
		if !f.Changed && v.IsSet(flagName) {
			val := v.Get(flagName)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		} else if f.Changed && !v.IsSet(flagName) {
			v.Set(flagName, f.Value)
		}
	})
}

func GetGracefulShutdownDuration() time.Duration {
	if !v.IsSet(gracefulShutdown) {
		return defaultGracefulShutdown
	}

	return v.GetDuration(gracefulShutdown)
}

func GetHttpRequestTimeout() time.Duration {
	return defaultHttpTimeout
}

func GetTargetNamespace() string {
	if !v.IsSet(namespace) {
		return defaultNamespace
	}

	return v.GetString(namespace)
}

func GetDeviceID() string {
	if !v.IsSet(deviceID) {
		id, err := machineid.ID()
		if err != nil {
			id = uuid.New().String()
		}

		// save id for the next call
		v.Set(deviceID, id)

		return id
	}

	return v.GetString(deviceID)
}

func GetCARootFile() string {
	return v.GetString(caRoot)
}

func GetCertificateFile() string {
	return v.GetString(certFile)
}

func GetPrivateKey() string {
	return v.GetString(privateKey)
}

func GetServerAddress() string {
	return v.GetString(server)
}

func GetRepoRetryConfig() RetryConfig {
	config := RetryConfig{
		// InitialInterval: parseDuration(repoRetryInitialInterval),
		// Multiplier:      viper.GetFloat64(repoRetryMultiplier),
		// MaxInterval:     parseDuration(repoRetryMaxInterval),
		// MaxElapsedTime:  parseDuration(repoRetryMaxElapsedTime),
	}

	return config
}
