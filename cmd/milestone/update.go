package milestone

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/shuyahonda/crm-cli/internal/crm"
)

var (
	updateName        string
	updateDescription string
	updateDate        string
)

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an existing milestone",
	Long:  "Update the name, description, or date of a milestone.",
	Args:  cobra.ExactArgs(1),
	Example: `  crm-cli milestone update 00000000-0000-0000-0000-000000000000 --name "Phase 1 Done"
  crm-cli milestone update 00000000-0000-0000-0000-000000000000 --date 2026-07-15`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		svc, err := newService(ctx)
		if err != nil {
			return err
		}

		input := crm.UpdateMilestoneInput{}

		if cmd.Flags().Changed("name") {
			input.Name = &updateName
		}
		if cmd.Flags().Changed("description") {
			input.Description = &updateDescription
		}
		if cmd.Flags().Changed("date") {
			t, err := time.Parse("2006-01-02", updateDate)
			if err != nil {
				return fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
			}
			input.Date = &t
		}

		if err := svc.Update(ctx, args[0], input); err != nil {
			return err
		}

		fmt.Printf("Milestone %s updated.\n", args[0])
		return nil
	},
}

func init() {
	updateCmd.Flags().StringVarP(&updateName, "name", "n", "", "New milestone name")
	updateCmd.Flags().StringVarP(&updateDescription, "description", "d", "", "New description")
	updateCmd.Flags().StringVar(&updateDate, "date", "", "New milestone date (YYYY-MM-DD)")
}
