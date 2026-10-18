package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

func testServer() *Server {
	return &Server{
		log: logging.Sugar{},
		cfg: &config.Config{},
	}
}

func postMCP(t *testing.T, srv *Server, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	srv.HandleRequest(rr, req)
	return rr
}

func decodeRPC(t *testing.T, rr *httptest.ResponseRecorder) MCPResponse {
	t.Helper()
	var resp MCPResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, rr.Body.String())
	}
	return resp
}

func TestHandleRequest_initialize(t *testing.T) {
	rr := postMCP(t, testServer(), `{"jsonrpc":"2.0","method":"initialize","id":1}`, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	resp := decodeRPC(t, rr)
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result type %T", resp.Result)
	}
	if result["protocolVersion"] != ProtocolVersion {
		t.Errorf("protocolVersion = %v", result["protocolVersion"])
	}
	info := result["serverInfo"].(map[string]interface{})
	if info["name"] != "TicketMosquito" {
		t.Errorf("name = %v", info["name"])
	}
}

func TestHandleRequest_toolsList(t *testing.T) {
	rr := postMCP(t, testServer(), `{"jsonrpc":"2.0","method":"tools/list","id":2}`, nil)
	resp := decodeRPC(t, rr)
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	result := resp.Result.(map[string]interface{})
	tools := result["tools"].([]interface{})
	names := map[string]bool{}
	for _, raw := range tools {
		m := raw.(map[string]interface{})
		names[m["name"].(string)] = true
	}
	for _, want := range []string{"jira_search", "jira_get_issue", "jira_list_projects"} {
		if !names[want] {
			t.Errorf("missing tool %s", want)
		}
	}
}

func TestHandleRequest_ping(t *testing.T) {
	rr := postMCP(t, testServer(), `{"jsonrpc":"2.0","method":"ping","id":3}`, nil)
	resp := decodeRPC(t, rr)
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
}

func TestHandleRequest_unknownMethod(t *testing.T) {
	rr := postMCP(t, testServer(), `{"jsonrpc":"2.0","method":"nope","id":4}`, nil)
	resp := decodeRPC(t, rr)
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Fatalf("expected -32601, got %+v", resp.Error)
	}
}

func TestHandleRequest_forbiddenOrigin(t *testing.T) {
	rr := postMCP(t, testServer(), `{"jsonrpc":"2.0","method":"ping","id":1}`, map[string]string{
		"Origin": "https://evil.example",
	})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestHandleRequest_localhostOrigin(t *testing.T) {
	rr := postMCP(t, testServer(), `{"jsonrpc":"2.0","method":"ping","id":1}`, map[string]string{
		"Origin": "http://127.0.0.1:3000",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}
