package main

import (
	"github.com/shuyahonda/crm-cli/cmd"
	milestoneCmd "github.com/shuyahonda/crm-cli/cmd/milestone"
	mcpCmd "github.com/shuyahonda/crm-cli/cmd/mcp"
)

func main() {
	cmd.Execute()
}

func init() {
	cmd.RegisterSubcommands(milestoneCmd.Cmd, mcpCmd.Cmd)
}
