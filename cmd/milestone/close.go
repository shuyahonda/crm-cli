package milestone

import (
	"fmt"

	"github.com/spf13/cobra"
)

var closeCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Close (deactivate) a milestone",
	Long:  "Mark a milestone as closed/inactive in Dynamics 365 CRM.",
	Args:  cobra.ExactArgs(1),
	Example: `  crm-cli milestone close 00000000-0000-0000-0000-000000000000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		svc, err := newService(ctx)
		if err != nil {
			return err
		}

		if err := svc.Close(ctx, args[0]); err != nil {
			return err
		}

		fmt.Printf("Milestone %s closed.\n", args[0])
		return nil
	},
}

var reopenCmd = &cobra.Command{
	Use:   "reopen <id>",
	Short: "Reopen (reactivate) a milestone",
	Long:  "Reopen a previously closed milestone in Dynamics 365 CRM.",
	Args:  cobra.ExactArgs(1),
	Example: `  crm-cli milestone reopen 00000000-0000-0000-0000-000000000000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		svc, err := newService(ctx)
		if err != nil {
			return err
		}

		if err := svc.Reopen(ctx, args[0]); err != nil {
			return err
		}

		fmt.Printf("Milestone %s reopened.\n", args[0])
		return nil
	},
}
