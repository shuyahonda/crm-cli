package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shuyahonda/crm-cli/internal/crm"
)

// ToolHandler is a function that handles a tool call and returns text content.
type ToolHandler func(ctx context.Context, args json.RawMessage) (string, error)

// toolRegistry maps tool names to their definitions and handlers.
type toolRegistry struct {
	tools    []Tool
	handlers map[string]ToolHandler
}

// newToolRegistry creates a registry with all milestone management tools registered.
func newToolRegistry(svc *crm.MilestoneService) *toolRegistry {
	r := &toolRegistry{
		handlers: make(map[string]ToolHandler),
	}

	// --- milestone_list ---
	r.register(Tool{
		Name:        "milestone_list",
		Description: "List milestones in Dynamics 365 CRM. Optionally filter by project ID.",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"project_id": {
					Type:        "string",
					Description: "Filter milestones by project GUID (optional)",
				},
			},
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		var p struct {
			ProjectID string `json:"project_id"`
		}
		_ = json.Unmarshal(args, &p)

		milestones, err := svc.List(ctx, p.ProjectID)
		if err != nil {
			return "", err
		}
		if len(milestones) == 0 {
			return "No milestones found.", nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d milestone(s):\n\n", len(milestones)))
		for _, m := range milestones {
			sb.WriteString(fmt.Sprintf("- ID: %s\n  Name: %s\n  Date: %s\n  State: %s\n\n",
				m.ID, m.Name, m.DateStr(), m.StateLabel()))
		}
		return sb.String(), nil
	})

	// --- milestone_get ---
	r.register(Tool{
		Name:        "milestone_get",
		Description: "Get details of a specific milestone by its ID.",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"id": {
					Type:        "string",
					Description: "Milestone GUID",
				},
			},
			Required: []string{"id"},
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if p.ID == "" {
			return "", fmt.Errorf("id is required")
		}

		m, err := svc.Get(ctx, p.ID)
		if err != nil {
			return "", err
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("ID:          %s\n", m.ID))
		sb.WriteString(fmt.Sprintf("Name:        %s\n", m.Name))
		sb.WriteString(fmt.Sprintf("Description: %s\n", m.Description))
		sb.WriteString(fmt.Sprintf("Date:        %s\n", m.DateStr()))
		sb.WriteString(fmt.Sprintf("State:       %s\n", m.StateLabel()))
		if m.ProjectID != "" {
			sb.WriteString(fmt.Sprintf("Project ID:  %s\n", m.ProjectID))
		}
		return sb.String(), nil
	})

	// --- milestone_create ---
	r.register(Tool{
		Name:        "milestone_create",
		Description: "Create a new milestone in Dynamics 365 CRM.",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"name": {
					Type:        "string",
					Description: "Milestone name (required)",
				},
				"description": {
					Type:        "string",
					Description: "Milestone description",
				},
				"date": {
					Type:        "string",
					Description: "Milestone target date in YYYY-MM-DD format",
				},
				"project_id": {
					Type:        "string",
					Description: "Project GUID to associate this milestone with",
				},
			},
			Required: []string{"name"},
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		var p struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Date        string `json:"date"`
			ProjectID   string `json:"project_id"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if p.Name == "" {
			return "", fmt.Errorf("name is required")
		}

		input := crm.CreateMilestoneInput{
			Name:        p.Name,
			Description: p.Description,
			ProjectID:   p.ProjectID,
		}
		if p.Date != "" {
			t, err := time.Parse("2006-01-02", p.Date)
			if err != nil {
				return "", fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
			}
			input.Date = &t
		}

		created, err := svc.Create(ctx, input)
		if err != nil {
			return "", err
		}

		if created.ID != "" {
			return fmt.Sprintf("Milestone created successfully.\nID: %s\nName: %s", created.ID, created.Name), nil
		}
		return fmt.Sprintf("Milestone '%s' created successfully.", p.Name), nil
	})

	// --- milestone_update ---
	r.register(Tool{
		Name:        "milestone_update",
		Description: "Update an existing milestone's name, description, or date.",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"id": {
					Type:        "string",
					Description: "Milestone GUID to update",
				},
				"name": {
					Type:        "string",
					Description: "New milestone name",
				},
				"description": {
					Type:        "string",
					Description: "New description",
				},
				"date": {
					Type:        "string",
					Description: "New target date in YYYY-MM-DD format",
				},
			},
			Required: []string{"id"},
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		var p struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Date        string `json:"date"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if p.ID == "" {
			return "", fmt.Errorf("id is required")
		}

		input := crm.UpdateMilestoneInput{}
		if p.Name != "" {
			input.Name = &p.Name
		}
		if p.Description != "" {
			input.Description = &p.Description
		}
		if p.Date != "" {
			t, err := time.Parse("2006-01-02", p.Date)
			if err != nil {
				return "", fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
			}
			input.Date = &t
		}

		if err := svc.Update(ctx, p.ID, input); err != nil {
			return "", err
		}

		return fmt.Sprintf("Milestone %s updated successfully.", p.ID), nil
	})

	// --- milestone_close ---
	r.register(Tool{
		Name:        "milestone_close",
		Description: "Close (deactivate) a milestone to mark it as completed.",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"id": {
					Type:        "string",
					Description: "Milestone GUID to close",
				},
			},
			Required: []string{"id"},
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if p.ID == "" {
			return "", fmt.Errorf("id is required")
		}

		if err := svc.Close(ctx, p.ID); err != nil {
			return "", err
		}

		return fmt.Sprintf("Milestone %s has been closed.", p.ID), nil
	})

	// --- milestone_reopen ---
	r.register(Tool{
		Name:        "milestone_reopen",
		Description: "Reopen (reactivate) a previously closed milestone.",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"id": {
					Type:        "string",
					Description: "Milestone GUID to reopen",
				},
			},
			Required: []string{"id"},
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if p.ID == "" {
			return "", fmt.Errorf("id is required")
		}

		if err := svc.Reopen(ctx, p.ID); err != nil {
			return "", err
		}

		return fmt.Sprintf("Milestone %s has been reopened.", p.ID), nil
	})

	// --- milestone_delete ---
	r.register(Tool{
		Name:        "milestone_delete",
		Description: "Permanently delete a milestone from Dynamics 365 CRM. This cannot be undone.",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"id": {
					Type:        "string",
					Description: "Milestone GUID to delete",
				},
			},
			Required: []string{"id"},
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if p.ID == "" {
			return "", fmt.Errorf("id is required")
		}

		if err := svc.Delete(ctx, p.ID); err != nil {
			return "", err
		}

		return fmt.Sprintf("Milestone %s has been permanently deleted.", p.ID), nil
	})

	return r
}

// register adds a tool and its handler to the registry.
func (r *toolRegistry) register(tool Tool, handler ToolHandler) {
	r.tools = append(r.tools, tool)
	r.handlers[tool.Name] = handler
}

// call dispatches a tool call by name.
func (r *toolRegistry) call(ctx context.Context, name string, args json.RawMessage) (string, error) {
	handler, ok := r.handlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, args)
}
