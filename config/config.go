package config

import (
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var Module = fx.Provide(
	NewAppConfig,
	NewDebugConfig,
	NewGrpcConfig,
	NewAMQPConfig,
	NewMonitoringConfig,
)

type (
	AppConfig struct {
		API        APIConfig
		Debug      DebugConfig
		AMQP       AMQPConfig
		Monitoring MonitoringConfig
	}
	APIConfig struct {
		GRPC GRPCConfig
	}
	DebugConfig struct {
		Debug bool
	}
	GRPCConfig struct {
		Host string
		Port uint
	}
	AMQPConfig struct {
		Host     string
		Port     uint
		User     string
		Password string
	}
	MonitoringConfig struct {
		Host string
		Port uint16
	}
)

func NewAppConfig() (AppConfig, error) {
	conf := viper.New()
	config := new(AppConfig)

	conf.SetConfigFile("config.yaml")
	conf.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	conf.AutomaticEnv()

	err := conf.ReadInConfig()
	if err != nil {
		return AppConfig{}, err
	}

	err = conf.Unmarshal(config)

	return *config, err
}

func NewDebugConfig(config AppConfig) DebugConfig {
	return config.Debug
}

func NewGrpcConfig(config AppConfig) GRPCConfig {
	return config.API.GRPC
}

func NewAMQPConfig(config AppConfig) AMQPConfig {
	return config.AMQP
}

func NewMonitoringConfig(config AppConfig) MonitoringConfig {
	return config.Monitoring
}
