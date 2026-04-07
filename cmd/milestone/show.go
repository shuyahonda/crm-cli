package milestone

import (
	"fmt"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show details of a milestone",
	Long:  "Show detailed information about a specific milestone.",
	Args:  cobra.ExactArgs(1),
	Example: `  crm-cli milestone show 00000000-0000-0000-0000-000000000000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		svc, err := newService(ctx)
		if err != nil {
			return err
		}

		m, err := svc.Get(ctx, args[0])
		if err != nil {
			return err
		}

		fmt.Printf("ID:          %s\n", m.ID)
		fmt.Printf("Name:        %s\n", m.Name)
		fmt.Printf("Description: %s\n", m.Description)
		fmt.Printf("Date:        %s\n", m.DateStr())
		fmt.Printf("State:       %s\n", m.StateLabel())
		if m.ProjectID != "" {
			fmt.Printf("Project ID:  %s\n", m.ProjectID)
		}
		if m.CreatedOn != nil {
			fmt.Printf("Created:     %s\n", m.CreatedOn.Local().Format("2006-01-02 15:04:05"))
		}
		if m.ModifiedOn != nil {
			fmt.Printf("Modified:    %s\n", m.ModifiedOn.Local().Format("2006-01-02 15:04:05"))
		}

		return nil
	},
}
