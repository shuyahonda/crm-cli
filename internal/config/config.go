package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all configuration for the CRM CLI tool.
type Config struct {
	// Azure AD / Entra ID settings
	TenantID     string `mapstructure:"tenant_id"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`

	// Dynamics 365 settings
	CRMBaseURL string `mapstructure:"crm_base_url"` // e.g. https://yourorg.crm.dynamics.com
	APIVersion string `mapstructure:"api_version"`  // default: 9.2

	// Auth flow: "client_credentials" or "device_code"
	AuthFlow string `mapstructure:"auth_flow"`
}

const defaultAPIVersion = "9.2"
const defaultAuthFlow = "client_credentials"

// Load reads configuration from the config file and environment variables.
// Priority: env vars > config file > defaults.
func Load() (*Config, error) {
	viper.SetConfigName("crm-cli")
	viper.SetConfigType("yaml")

	// Search paths
	viper.AddConfigPath(".")
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(filepath.Join(home, ".config", "crm-cli"))
	}

	// Environment variable mapping
	viper.SetEnvPrefix("CRM")
	viper.AutomaticEnv()

	// Explicit env var bindings (CRM_TENANT_ID, CRM_CLIENT_ID, etc.)
	_ = viper.BindEnv("tenant_id", "CRM_TENANT_ID")
	_ = viper.BindEnv("client_id", "CRM_CLIENT_ID")
	_ = viper.BindEnv("client_secret", "CRM_CLIENT_SECRET")
	_ = viper.BindEnv("crm_base_url", "CRM_BASE_URL")
	_ = viper.BindEnv("api_version", "CRM_API_VERSION")
	_ = viper.BindEnv("auth_flow", "CRM_AUTH_FLOW")

	// Defaults
	viper.SetDefault("api_version", defaultAPIVersion)
	viper.SetDefault("auth_flow", defaultAuthFlow)

	// Try to read config file (not fatal if missing)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

// Validate checks that required fields are set.
func (c *Config) Validate() error {
	if c.TenantID == "" {
		return fmt.Errorf("tenant_id is required (set CRM_TENANT_ID env var or config file)")
	}
	if c.ClientID == "" {
		return fmt.Errorf("client_id is required (set CRM_CLIENT_ID env var or config file)")
	}
	if c.CRMBaseURL == "" {
		return fmt.Errorf("crm_base_url is required (set CRM_BASE_URL env var or config file)")
	}
	if c.AuthFlow == "client_credentials" && c.ClientSecret == "" {
		return fmt.Errorf("client_secret is required for client_credentials auth flow (set CRM_CLIENT_SECRET)")
	}
	return nil
}

// WebAPIBaseURL returns the Dynamics 365 Web API base URL.
func (c *Config) WebAPIBaseURL() string {
	return fmt.Sprintf("%s/api/data/v%s", c.CRMBaseURL, c.APIVersion)
}
