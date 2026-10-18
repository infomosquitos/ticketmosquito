package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/search"
	"github.com/infomosquitos/ticketmosquito/internal/store"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// ProtocolVersion is the Streamable HTTP revision we advertise.
const ProtocolVersion = "2025-03-26"

// Server is the MCP Streamable HTTP endpoint.
type Server struct {
	db  store.Store
	cfg *config.Config
	log logging.Sugar
}

// MCPRequest is a JSON-RPC 2.0 request.
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// MCPResponse is a JSON-RPC 2.0 response.
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

// MCPError is a JSON-RPC error object.
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// Tool describes one MCP tool.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// New creates an MCP server over the local catalog.
func New(db store.Store, cfg *config.Config, log logging.Sugar) *Server {
	return &Server{db: db, cfg: cfg, log: log}
}

// HandleRequest serves Streamable HTTP on POST /mcp (JSON-RPC in, JSON out).
func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if !originAllowed(r.Header.Get("Origin")) {
		writeJSONRPCError(w, http.StatusForbidden, nil, -32000, "forbidden origin", "")
		return
	}

	if r.Method == http.MethodGet {
		w.Header().Set("Allow", "POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	if !acceptOK(r.Header.Get("Accept")) {
		writeJSONRPCError(w, http.StatusBadRequest, nil, -32600, "Accept must include application/json and text/event-stream", "")
		return
	}

	if ver := r.Header.Get("MCP-Protocol-Version"); ver != "" && !protocolVersionOK(ver) {
		writeJSONRPCError(w, http.StatusBadRequest, nil, -32600, "unsupported MCP-Protocol-Version", ver)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONRPCError(w, http.StatusBadRequest, nil, -32700, "failed to read body", "")
		return
	}
	defer r.Body.Close()

	var req MCPRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONRPCError(w, http.StatusBadRequest, nil, -32700, "parse error", "invalid JSON")
		return
	}

	if s.log.Enabled() {
		s.log.Infow("MCP request received", "method", req.Method)
	}

	if req.ID == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	resp := s.dispatch(&req)
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) dispatch(req *MCPRequest) *MCPResponse {
	switch req.Method {
	case "initialize":
		return okResult(req.ID, map[string]interface{}{
			"protocolVersion": ProtocolVersion,
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "TicketMosquito",
				"version": "0.1.0",
			},
		})
	case "notifications/initialized":
		return nil
	case "tools/list":
		return okResult(req.ID, map[string]interface{}{"tools": toolDefinitions()})
	case "tools/call":
		return s.handleToolsCall(req.Params, req.ID)
	case "ping":
		return okResult(req.ID, map[string]string{"status": "ok"})
	default:
		return rpcError(req.ID, -32601, "method not found", req.Method)
	}
}

func toolDefinitions() []Tool {
	return []Tool{
		{
			Name:        "jira_search",
			Description: "Search local Jira issues using BM25/FTS lexical search. Results include key for use with jira_get_issue.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Search query (multi-word AND)"},
					"project": {"type": "string", "description": "Optional project key to filter"},
					"limit": {"type": "integer", "description": "Max results (default 10)"}
				},
				"required": ["query"]
			}`),
		},
		{
			Name:        "jira_list_projects",
			Description: "List all crawled Jira projects in the local catalog",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {},
				"required": []
			}`),
		},
		{
			Name:        "jira_get_issue",
			Description: "Get a crawled Jira issue by key (e.g. PROJ-123). Returns Markdown content and metadata from the local catalog.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"issue_key": {"type": "string", "description": "Issue key (PROJ-123)"}
				},
				"required": ["issue_key"]
			}`),
		},
	}
}

func (s *Server) handleToolsCall(params json.RawMessage, id interface{}) *MCPResponse {
	var callParams map[string]interface{}
	if err := json.Unmarshal(params, &callParams); err != nil {
		return rpcError(id, -32602, "invalid params", err.Error())
	}

	toolName, ok := callParams["name"].(string)
	if !ok {
		return rpcError(id, -32602, "invalid params", "tool name is required")
	}

	args, _ := callParams["arguments"].(map[string]interface{})
	if args == nil {
		args = make(map[string]interface{})
	}

	var result interface{}
	var err error
	start := time.Now()

	switch toolName {
	case "jira_search":
		result, err = s.toolSearch(args)
	case "jira_list_projects":
		result, err = s.toolListProjects()
	case "jira_get_issue":
		result, err = s.toolGetIssue(args)
	default:
		err = fmt.Errorf("unknown tool: %s", toolName)
	}

	if s.log.Enabled() {
		s.log.Infow("tool executed",
			"tool", toolName,
			"duration_ms", time.Since(start).Milliseconds(),
			"success", err == nil)
	}

	if err != nil {
		return okResult(id, map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": fmt.Sprintf("Error: %v", err)},
			},
			"isError": true,
		})
	}

	resultStr, _ := json.MarshalIndent(result, "", "  ")
	return okResult(id, map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": string(resultStr)},
		},
	})
}

func (s *Server) toolSearch(args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required")
	}
	project, _ := args["project"].(string)
	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	results, err := s.db.SearchIssues(context.Background(), query, project, limit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	return search.ToSearchHits(results, s.cfg.MCP.ExposeInternalIDs), nil
}

func (s *Server) toolListProjects() (interface{}, error) {
	projects, err := s.db.ListProjects(context.Background())
	if err != nil {
		return nil, err
	}
	return search.ToProjectSummaries(projects), nil
}

func (s *Server) toolGetIssue(args map[string]interface{}) (interface{}, error) {
	key, _ := args["issue_key"].(string)
	if key == "" {
		key, _ = args["key"].(string)
	}
	if key == "" {
		return nil, fmt.Errorf("issue_key is required")
	}
	iss, err := s.db.GetIssueByKey(context.Background(), key)
	if errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("issue not found: %s", key)
	}
	if err != nil {
		return nil, err
	}
	return search.ToIssueDetail(iss, s.cfg.MCP.ExposeInternalIDs), nil
}

func okResult(id interface{}, result interface{}) *MCPResponse {
	return &MCPResponse{JSONRPC: "2.0", Result: result, ID: id}
}

func rpcError(id interface{}, code int, message, data string) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		Error:   &MCPError{Code: code, Message: message, Data: data},
		ID:      id,
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeJSONRPCError(w http.ResponseWriter, httpStatus int, id interface{}, code int, message, data string) {
	writeJSON(w, httpStatus, rpcError(id, code, message, data))
}

func acceptOK(accept string) bool {
	a := strings.ToLower(accept)
	if strings.Contains(a, "*/*") {
		return true
	}
	return strings.Contains(a, "application/json") && strings.Contains(a, "text/event-stream")
}

func protocolVersionOK(ver string) bool {
	switch ver {
	case "2025-03-26", "2025-06-18", "2025-11-25", "2026-07-28":
		return true
	default:
		return false
	}
}

func originAllowed(origin string) bool {
	if origin == "" || origin == "null" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
