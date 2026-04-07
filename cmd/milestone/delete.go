package milestone

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var deleteForce bool

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Permanently delete a milestone",
	Long:  "Permanently delete a milestone from Dynamics 365 CRM. This action cannot be undone.",
	Args:  cobra.ExactArgs(1),
	Example: `  crm-cli milestone delete 00000000-0000-0000-0000-000000000000
  crm-cli milestone delete 00000000-0000-0000-0000-000000000000 --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		id := args[0]

		if !deleteForce {
			fmt.Printf("Are you sure you want to permanently delete milestone %s? [y/N] ", id)
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if answer != "y" && answer != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		svc, err := newService(ctx)
		if err != nil {
			return err
		}

		if err := svc.Delete(ctx, id); err != nil {
			return err
		}

		fmt.Printf("Milestone %s deleted.\n", id)
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Skip confirmation prompt")
}
