package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd is the base command for the CLI.
var rootCmd = &cobra.Command{
	Use:   "crm-cli",
	Short: "Dynamics 365 CRM milestone management CLI",
	Long: `crm-cli is a CLI tool for managing milestones in Dynamics 365 CRM.

It can be used directly from the command line or as an MCP (Model Context Protocol)
server to integrate with GitHub Copilot for AI-assisted milestone management.

Environment variables:
  CRM_TENANT_ID      Azure AD Tenant ID
  CRM_CLIENT_ID      Azure AD Application (Client) ID
  CRM_CLIENT_SECRET  Azure AD Client Secret (for client_credentials flow)
  CRM_BASE_URL       Dynamics 365 base URL (e.g. https://yourorg.crm.dynamics.com)
  CRM_API_VERSION    Web API version (default: 9.2)
  CRM_AUTH_FLOW      Auth flow: client_credentials (default) or device_code`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// RegisterSubcommands adds sub-commands to the root command.
// Called from main.go to avoid import cycles.
func RegisterSubcommands(cmds ...*cobra.Command) {
	rootCmd.AddCommand(cmds...)
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.config/crm-cli/crm-cli.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}
}
