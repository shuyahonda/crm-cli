package milestone

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listProjectID string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List milestones",
	Long:  "List all milestones in Dynamics 365 CRM, optionally filtered by project.",
	Example: `  crm-cli milestone list
  crm-cli milestone list --project 00000000-0000-0000-0000-000000000000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		svc, err := newService(ctx)
		if err != nil {
			return err
		}

		milestones, err := svc.List(ctx, listProjectID)
		if err != nil {
			return err
		}

		if len(milestones) == 0 {
			fmt.Println("No milestones found.")
			return nil
		}

		printMilestones(milestones)
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listProjectID, "project", "", "Filter by project ID (GUID)")
}
