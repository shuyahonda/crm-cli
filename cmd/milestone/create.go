package milestone

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/shuyahonda/crm-cli/internal/crm"
)

var (
	createName        string
	createDescription string
	createDate        string
	createProjectID   string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new milestone",
	Long:  "Create a new milestone in Dynamics 365 CRM.",
	Example: `  crm-cli milestone create --name "Phase 1 Complete" --date 2026-06-30
  crm-cli milestone create --name "Go-Live" --date 2026-12-01 --project 00000000-0000-0000-0000-000000000000 --description "Production deployment"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		svc, err := newService(ctx)
		if err != nil {
			return err
		}

		input := crm.CreateMilestoneInput{
			Name:        createName,
			Description: createDescription,
			ProjectID:   createProjectID,
		}

		if createDate != "" {
			t, err := time.Parse("2006-01-02", createDate)
			if err != nil {
				return fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
			}
			input.Date = &t
		}

		created, err := svc.Create(ctx, input)
		if err != nil {
			return err
		}

		if created.ID != "" {
			fmt.Printf("Milestone created: %s\n", created.ID)
		} else {
			fmt.Println("Milestone created successfully.")
		}

		return nil
	},
}

func init() {
	createCmd.Flags().StringVarP(&createName, "name", "n", "", "Milestone name (required)")
	createCmd.Flags().StringVarP(&createDescription, "description", "d", "", "Milestone description")
	createCmd.Flags().StringVar(&createDate, "date", "", "Milestone date (YYYY-MM-DD)")
	createCmd.Flags().StringVar(&createProjectID, "project", "", "Project ID (GUID) to associate with")
	_ = createCmd.MarkFlagRequired("name")
}
