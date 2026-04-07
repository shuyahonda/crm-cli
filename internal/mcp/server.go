package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/shuyahonda/crm-cli/internal/auth"
	"github.com/shuyahonda/crm-cli/internal/config"
	"github.com/shuyahonda/crm-cli/internal/crm"
)

const (
	serverName    = "crm-milestone-mcp"
	serverVersion = "1.0.0"
	mcpVersion    = "2024-11-05"
)

// Server is the MCP server that exposes CRM milestone tools to AI assistants.
type Server struct {
	registry *toolRegistry
	in       *bufio.Scanner
	out      *json.Encoder
	logger   *log.Logger
}

// NewServer creates a new MCP server connected to stdin/stdout.
func NewServer(cfg *config.Config) *Server {
	authProvider := auth.NewProvider(cfg)
	crmClient := crm.NewClient(cfg, authProvider)
	milestoneSvc := crm.NewMilestoneService(crmClient)

	logger := log.New(os.Stderr, "[mcp] ", log.LstdFlags)

	return &Server{
		registry: newToolRegistry(milestoneSvc),
		in:       bufio.NewScanner(os.Stdin),
		out:      json.NewEncoder(os.Stdout),
		logger:   logger,
	}
}

// Serve starts the MCP server loop, reading JSON-RPC messages from stdin
// and writing responses to stdout.
func (s *Server) Serve(ctx context.Context) error {
	s.logger.Printf("CRM Milestone MCP server started (protocol %s)", mcpVersion)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !s.in.Scan() {
			if err := s.in.Err(); err != nil && err != io.EOF {
				return fmt.Errorf("stdin read error: %w", err)
			}
			// EOF = client disconnected
			s.logger.Println("Client disconnected.")
			return nil
		}

		line := s.in.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, ErrParseError, "Parse error", err.Error())
			continue
		}

		s.handleRequest(ctx, &req)
	}
}

// handleRequest dispatches an incoming JSON-RPC request.
func (s *Server) handleRequest(ctx context.Context, req *Request) {
	s.logger.Printf("-> %s (id=%v)", req.Method, req.ID)

	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "initialized":
		// Notification — no response required
	case "ping":
		s.sendResult(req.ID, map[string]any{})
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(ctx, req)
	default:
		s.sendError(req.ID, ErrMethodNotFound, "Method not found", req.Method)
	}
}

// handleInitialize responds to the MCP initialize handshake.
func (s *Server) handleInitialize(req *Request) {
	result := InitializeResult{
		ProtocolVersion: mcpVersion,
		Capabilities: map[string]any{
			"tools": map[string]any{},
		},
		ServerInfo: ServerInfo{
			Name:    serverName,
			Version: serverVersion,
		},
	}
	s.sendResult(req.ID, result)
}

// handleToolsList returns the list of available tools.
func (s *Server) handleToolsList(req *Request) {
	s.sendResult(req.ID, ToolsListResult{Tools: s.registry.tools})
}

// handleToolsCall executes a tool and returns the result.
func (s *Server) handleToolsCall(ctx context.Context, req *Request) {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, ErrInvalidParams, "Invalid params", err.Error())
		return
	}

	s.logger.Printf("   tool=%s args=%s", params.Name, string(params.Arguments))

	text, err := s.registry.call(ctx, params.Name, params.Arguments)
	if err != nil {
		s.logger.Printf("   tool error: %v", err)
		s.sendResult(req.ID, ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error: %s", err.Error())}},
			IsError: true,
		})
		return
	}

	s.logger.Printf("   tool success: %d chars", len(text))
	s.sendResult(req.ID, ToolCallResult{
		Content: []ToolContent{{Type: "text", Text: text}},
	})
}

// sendResult sends a successful JSON-RPC response.
func (s *Server) sendResult(id any, result any) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	if err := s.out.Encode(resp); err != nil {
		s.logger.Printf("failed to write response: %v", err)
	}
}

// sendError sends an error JSON-RPC response.
func (s *Server) sendError(id any, code int, message, data string) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	if err := s.out.Encode(resp); err != nil {
		s.logger.Printf("failed to write error response: %v", err)
	}
}
