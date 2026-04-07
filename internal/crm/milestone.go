package crm

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// Milestone represents a Dynamics 365 milestone record.
// Entity: msdyn_milestone (Project Operations)
type Milestone struct {
	ID          string     `json:"msdyn_milestoneid,omitempty"`
	Name        string     `json:"msdyn_milestonename,omitempty"`
	Description string     `json:"msdyn_description,omitempty"`
	Date        *time.Time `json:"msdyn_milestonedate,omitempty"`
	StateCode   int        `json:"statecode,omitempty"`  // 0=Active, 1=Inactive
	StatusCode  int        `json:"statuscode,omitempty"` // 1=Active, 2=Inactive
	ProjectID   string     `json:"_msdyn_project_value,omitempty"`
	ProjectName string     `json:"msdyn_project@OData.Community.Display.V1.FormattedValue,omitempty"`
	CreatedOn   *time.Time `json:"createdon,omitempty"`
	ModifiedOn  *time.Time `json:"modifiedon,omitempty"`
}

// MilestoneState represents the state of a milestone.
const (
	MilestoneStateActive   = 0
	MilestoneStateInactive = 1

	MilestoneStatusActive   = 1
	MilestoneStatusInactive = 2
)

// CreateMilestoneInput holds fields for creating a new milestone.
type CreateMilestoneInput struct {
	Name        string
	Description string
	Date        *time.Time
	ProjectID   string // Dynamics 365 GUID of the related project
}

// UpdateMilestoneInput holds fields for updating a milestone.
type UpdateMilestoneInput struct {
	Name        *string
	Description *string
	Date        *time.Time
}

// MilestoneService provides operations on Dynamics 365 milestones.
type MilestoneService struct {
	client *Client
}

// NewMilestoneService creates a new MilestoneService.
func NewMilestoneService(client *Client) *MilestoneService {
	return &MilestoneService{client: client}
}

const milestoneEntity = "msdyn_milestones"

// List retrieves all milestones, optionally filtered by project ID.
func (s *MilestoneService) List(ctx context.Context, projectID string) ([]Milestone, error) {
	query := url.Values{}
	query.Set("$select", "msdyn_milestoneid,msdyn_milestonename,msdyn_description,msdyn_milestonedate,statecode,statuscode,createdon,modifiedon,_msdyn_project_value")
	query.Set("$orderby", "msdyn_milestonedate asc")

	if projectID != "" {
		query.Set("$filter", fmt.Sprintf("_msdyn_project_value eq '%s'", projectID))
	}

	path := milestoneEntity + "?" + query.Encode()

	var resp ODataResponse[Milestone]
	if err := s.client.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("failed to list milestones: %w", err)
	}

	return resp.Value, nil
}

// Get retrieves a single milestone by ID.
func (s *MilestoneService) Get(ctx context.Context, id string) (*Milestone, error) {
	query := url.Values{}
	query.Set("$select", "msdyn_milestoneid,msdyn_milestonename,msdyn_description,msdyn_milestonedate,statecode,statuscode,createdon,modifiedon,_msdyn_project_value")

	path := fmt.Sprintf("%s(%s)?%s", milestoneEntity, id, query.Encode())

	var milestone Milestone
	if err := s.client.get(ctx, path, &milestone); err != nil {
		return nil, fmt.Errorf("failed to get milestone %s: %w", id, err)
	}

	return &milestone, nil
}

// Create creates a new milestone record.
func (s *MilestoneService) Create(ctx context.Context, input CreateMilestoneInput) (*Milestone, error) {
	body := map[string]any{
		"msdyn_milestonename": input.Name,
	}

	if input.Description != "" {
		body["msdyn_description"] = input.Description
	}
	if input.Date != nil {
		body["msdyn_milestonedate"] = input.Date.UTC().Format(time.RFC3339)
	}
	if input.ProjectID != "" {
		body["msdyn_project@odata.bind"] = fmt.Sprintf("/msdyn_projects(%s)", input.ProjectID)
	}

	var created Milestone
	if err := s.client.post(ctx, milestoneEntity, body, &created); err != nil {
		return nil, fmt.Errorf("failed to create milestone: %w", err)
	}

	// If the response doesn't include the ID (some D365 versions return it in the header),
	// we return what we have.
	return &created, nil
}

// Update updates an existing milestone.
func (s *MilestoneService) Update(ctx context.Context, id string, input UpdateMilestoneInput) error {
	body := map[string]any{}

	if input.Name != nil {
		body["msdyn_milestonename"] = *input.Name
	}
	if input.Description != nil {
		body["msdyn_description"] = *input.Description
	}
	if input.Date != nil {
		body["msdyn_milestonedate"] = input.Date.UTC().Format(time.RFC3339)
	}

	if len(body) == 0 {
		return fmt.Errorf("no fields to update")
	}

	path := fmt.Sprintf("%s(%s)", milestoneEntity, id)
	if err := s.client.patch(ctx, path, body); err != nil {
		return fmt.Errorf("failed to update milestone %s: %w", id, err)
	}

	return nil
}

// Close deactivates a milestone (sets state to Inactive).
func (s *MilestoneService) Close(ctx context.Context, id string) error {
	body := map[string]any{
		"statecode":  MilestoneStateInactive,
		"statuscode": MilestoneStatusInactive,
	}

	path := fmt.Sprintf("%s(%s)", milestoneEntity, id)
	if err := s.client.patch(ctx, path, body); err != nil {
		return fmt.Errorf("failed to close milestone %s: %w", id, err)
	}

	return nil
}

// Reopen reactivates a milestone (sets state to Active).
func (s *MilestoneService) Reopen(ctx context.Context, id string) error {
	body := map[string]any{
		"statecode":  MilestoneStateActive,
		"statuscode": MilestoneStatusActive,
	}

	path := fmt.Sprintf("%s(%s)", milestoneEntity, id)
	if err := s.client.patch(ctx, path, body); err != nil {
		return fmt.Errorf("failed to reopen milestone %s: %w", id, err)
	}

	return nil
}

// Delete permanently deletes a milestone.
func (s *MilestoneService) Delete(ctx context.Context, id string) error {
	path := fmt.Sprintf("%s(%s)", milestoneEntity, id)
	if err := s.client.delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete milestone %s: %w", id, err)
	}
	return nil
}

// StateLabel returns a human-readable state label.
func (m *Milestone) StateLabel() string {
	if m.StateCode == MilestoneStateActive {
		return "Active"
	}
	return "Closed"
}

// DateStr returns the milestone date as a formatted string.
func (m *Milestone) DateStr() string {
	if m.Date == nil {
		return "(no date)"
	}
	return m.Date.Local().Format("2006-01-02")
}
