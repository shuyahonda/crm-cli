package mcp

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/shuyahonda/crm-cli/internal/config"
	"github.com/shuyahonda/crm-cli/internal/mcp"
)

// Cmd is the "mcp serve" command.
var Cmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP server for GitHub Copilot integration",
	Long: `Start the Model Context Protocol (MCP) server.

This server exposes CRM milestone management as tools that can be called
by AI assistants such as GitHub Copilot via the MCP protocol (JSON-RPC 2.0
over stdin/stdout).

To configure GitHub Copilot to use this server, add the following to your
VS Code settings.json or .copilot/mcp.json:

  {
    "mcpServers": {
      "crm-milestone": {
        "command": "crm-cli",
        "args": ["mcp"],
        "env": {
          "CRM_TENANT_ID": "<your-tenant-id>",
          "CRM_CLIENT_ID": "<your-client-id>",
          "CRM_CLIENT_SECRET": "<your-client-secret>",
          "CRM_BASE_URL": "https://<yourorg>.crm.dynamics.com"
        }
      }
    }
  }

Available tools:
  milestone_list    - List milestones (optionally filter by project)
  milestone_get     - Get details of a specific milestone
  milestone_create  - Create a new milestone
  milestone_update  - Update a milestone's name, description, or date
  milestone_close   - Close (deactivate) a milestone
  milestone_reopen  - Reopen a previously closed milestone
  milestone_delete  - Permanently delete a milestone`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
			return err
		}
		if err := cfg.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
			return err
		}

		server := mcp.NewServer(cfg)
		return server.Serve(ctx)
	},
}
