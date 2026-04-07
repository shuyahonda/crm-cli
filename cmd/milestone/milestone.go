package milestone

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/shuyahonda/crm-cli/internal/auth"
	"github.com/shuyahonda/crm-cli/internal/config"
	"github.com/shuyahonda/crm-cli/internal/crm"
)

// Cmd is the parent "milestone" command.
var Cmd = &cobra.Command{
	Use:   "milestone",
	Short: "Manage milestones in Dynamics 365 CRM",
	Long:  "Create, list, show, update, close, reopen, and delete milestones in Dynamics 365 CRM.",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(closeCmd)
	Cmd.AddCommand(reopenCmd)
	Cmd.AddCommand(deleteCmd)
}

// newService creates a MilestoneService from the current config.
func newService(ctx context.Context) (*crm.MilestoneService, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	authProvider := auth.NewProvider(cfg)
	client := crm.NewClient(cfg, authProvider)
	return crm.NewMilestoneService(client), nil
}

// printMilestones prints a table of milestones.
func printMilestones(milestones []crm.Milestone) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tDATE\tSTATE\tPROJECT")
	fmt.Fprintln(w, "--\t----\t----\t-----\t-------")
	for _, m := range milestones {
		project := m.ProjectName
		if project == "" && m.ProjectID != "" {
			project = m.ProjectID[:8] + "..."
		}
		if project == "" {
			project = "(none)"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			m.ID, m.Name, m.DateStr(), m.StateLabel(), project)
	}
	w.Flush()
}
