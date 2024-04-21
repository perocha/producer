package config

import (
	"os"

	"github.com/perocha/goutils/pkg/config"
)

type MicroserviceConfig struct {
	configClient                  *config.Config
	EventHubName                  string
	EventHubConnectionString      string
	AppInsightsInstrumentationKey string
}

// Initialize configuration client, either from environment variable or from file
func InitializeConfig() (*MicroserviceConfig, error) {
	// First try to get the App Configuration connection string as environment variable
	connectionString := os.Getenv("APPCONFIGURATION_CONNECTION_STRING")

	if connectionString != "" {
		mycfg, err := config.NewConfigFromConnectionString(connectionString)
		if err != nil {
			return nil, err
		}
		return &MicroserviceConfig{
			configClient: mycfg,
		}, nil
	} else {
		fileName := "config.yaml"
		mycfg, err := config.NewConfigFromFile(fileName)
		if err != nil {
			return nil, err
		}
		return &MicroserviceConfig{
			configClient: mycfg,
		}, nil
	}
}

// Refresh configuration, with the latest values from the configuration store
func (cfg *MicroserviceConfig) RefreshConfig() error {
	configValue, err := cfg.configClient.GetVar("APPINSIGHTS_INSTRUMENTATIONKEY")
	if err != nil {
		return err
	}
	cfg.AppInsightsInstrumentationKey = configValue

	configValue, err = cfg.configClient.GetVar("EVENTHUB_NAME")
	if err != nil {
		return err
	}
	cfg.EventHubName = configValue

	configValue, err = cfg.configClient.GetVar("EVENTHUB_PUBLISHER_CONNECTION_STRING")
	if err != nil {
		return err
	}
	cfg.EventHubConnectionString = configValue

	return nil
}
