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
	// TenantID can be a GUID, your domain (contoso.onmicrosoft.com), or "common".
	TenantID string `mapstructure:"tenant_id"`

	// ClientID is optional. If omitted, the well-known Dynamics CRM public
	// client ID (9cee029c-6210-4654-90bb-17e6e9d36617) is used, which does NOT
	// require any app registration by an admin.
	ClientID string `mapstructure:"client_id"`

	// ClientSecret is only required for the client_credentials auth flow.
	ClientSecret string `mapstructure:"client_secret"`

	// Username and Password are used by the "password" (ROPC) auth flow.
	// Only suitable for environments without MFA enforcement.
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`

	// Dynamics 365 settings
	CRMBaseURL string `mapstructure:"crm_base_url"` // e.g. https://yourorg.crm.dynamics.com
	APIVersion string `mapstructure:"api_version"`  // default: 9.2

	// Auth flow:
	//   device_code        - browser login, no app registration required (default)
	//   password           - username/password (ROPC), no app registration required
	//   client_credentials - service account, requires app registration + secret
	AuthFlow string `mapstructure:"auth_flow"`
}

// WellKnownDynamicsCRMClientID is Microsoft's pre-registered public client for
// Dynamics 365 SDK/PowerShell access. It supports device_code and password
// flows without requiring your own app registration.
const WellKnownDynamicsCRMClientID = "9cee029c-6210-4654-90bb-17e6e9d36617"

const defaultAPIVersion = "9.2"
const defaultAuthFlow = "device_code"

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

	_ = viper.BindEnv("tenant_id", "CRM_TENANT_ID")
	_ = viper.BindEnv("client_id", "CRM_CLIENT_ID")
	_ = viper.BindEnv("client_secret", "CRM_CLIENT_SECRET")
	_ = viper.BindEnv("username", "CRM_USERNAME")
	_ = viper.BindEnv("password", "CRM_PASSWORD")
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

	// Apply well-known client ID if none provided
	if cfg.ClientID == "" {
		cfg.ClientID = WellKnownDynamicsCRMClientID
	}

	return &cfg, nil
}

// Validate checks that required fields are set.
func (c *Config) Validate() error {
	if c.CRMBaseURL == "" {
		return fmt.Errorf("crm_base_url is required (set CRM_BASE_URL env var or config file)\n" +
			"  Example: CRM_BASE_URL=https://yourorg.crm.dynamics.com")
	}

	switch c.AuthFlow {
	case "device_code":
		// Only needs tenant_id (can be "common") and client_id (defaulted above)
		if c.TenantID == "" {
			c.TenantID = "common"
		}
	case "password":
		if c.Username == "" {
			return fmt.Errorf("username is required for password auth flow (set CRM_USERNAME)")
		}
		if c.Password == "" {
			return fmt.Errorf("password is required for password auth flow (set CRM_PASSWORD)")
		}
		if c.TenantID == "" {
			c.TenantID = "common"
		}
	case "client_credentials":
		if c.TenantID == "" {
			return fmt.Errorf("tenant_id is required for client_credentials auth flow (set CRM_TENANT_ID)")
		}
		if c.ClientSecret == "" {
			return fmt.Errorf("client_secret is required for client_credentials auth flow (set CRM_CLIENT_SECRET)")
		}
	default:
		return fmt.Errorf("unsupported auth_flow %q (choose: device_code, password, client_credentials)", c.AuthFlow)
	}

	return nil
}

// WebAPIBaseURL returns the Dynamics 365 Web API base URL.
func (c *Config) WebAPIBaseURL() string {
	return fmt.Sprintf("%s/api/data/v%s", c.CRMBaseURL, c.APIVersion)
}
