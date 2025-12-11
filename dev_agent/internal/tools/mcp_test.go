package tools

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCallWithRetriesInjectsMetaTag(t *testing.T) {
	server := newMCPMetaServer(t)
	client := NewMCPClient(server.URL())
	client.client = server.Client()

	_, err := client.callWithRetries("tools/call", map[string]any{"name": "unit"}, 0, 1)
	if err != nil {
		t.Fatalf("callWithRetries returned error: %v", err)
	}

	req := server.NextRequest(t)
	assertMetaTag(t, req)
}

func TestMCPToolMethodsIncludeMeta(t *testing.T) {
	server := newMCPMetaServer(t)
	client := NewMCPClient(server.URL())
	client.client = server.Client()

	tests := []struct {
		name string
		call func(t *testing.T)
		tool string
	}{
		{
			name: "ParallelExplore",
			call: func(t *testing.T) {
				if _, err := client.ParallelExplore("proj", "parent", []string{"prompt"}, "agent", 2); err != nil {
					t.Fatalf("ParallelExplore returned error: %v", err)
				}
			},
			tool: "parallel_explore",
		},
		{
			name: "GetBranch",
			call: func(t *testing.T) {
				if _, err := client.GetBranch("branch-1"); err != nil {
					t.Fatalf("GetBranch returned error: %v", err)
				}
			},
			tool: "get_branch",
		},
		{
			name: "BranchReadFile",
			call: func(t *testing.T) {
				if _, err := client.BranchReadFile("branch-2", "/tmp/file"); err != nil {
					t.Fatalf("BranchReadFile returned error: %v", err)
				}
			},
			tool: "branch_read_file",
		},
		{
			name: "BranchOutput",
			call: func(t *testing.T) {
				if _, err := client.BranchOutput("branch-3", true); err != nil {
					t.Fatalf("BranchOutput returned error: %v", err)
				}
			},
			tool: "branch_output",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.call(t)
			req := server.NextRequest(t)
			assertMetaTag(t, req)
			params := requestParams(t, req)
			if got := params["name"]; got != tc.tool {
				t.Fatalf("expected tool %s, got %#v", tc.tool, got)
			}
		})
	}
}

type mcpMetaServer struct {
	server   *httptest.Server
	requests chan map[string]any
}

func newMCPMetaServer(t *testing.T) *mcpMetaServer {
	t.Helper()
	requests := make(chan map[string]any, 10)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading request body: %v", err)
		}
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.UseNumber()
		var payload map[string]any
		if err := decoder.Decode(&payload); err != nil {
			t.Fatalf("failed decoding request body: %v", err)
		}
		requests <- payload
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":{"structuredContent":{"ok":true}}}`))
	})
	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
	})
	return &mcpMetaServer{
		server:   server,
		requests: requests,
	}
}

func (s *mcpMetaServer) URL() string {
	return s.server.URL
}

func (s *mcpMetaServer) Client() *http.Client {
	return s.server.Client()
}

func (s *mcpMetaServer) NextRequest(t *testing.T) map[string]any {
	t.Helper()
	select {
	case req := <-s.requests:
		return req
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for MCP request")
		return nil
	}
}

func requestParams(t *testing.T, payload map[string]any) map[string]any {
	t.Helper()
	params, ok := payload["params"].(map[string]any)
	if !ok {
		t.Fatalf("payload missing params: %#v", payload)
	}
	return params
}

func assertMetaTag(t *testing.T, payload map[string]any) {
	t.Helper()
	params := requestParams(t, payload)
	meta, ok := params["_meta"].(map[string]any)
	if !ok {
		t.Fatalf("params missing _meta: %#v", params)
	}
	if len(meta) != 1 {
		t.Fatalf("expected single entry in _meta, got %#v", meta)
	}
	if got := meta["ai.tidb.pantheon-ai/agent"]; got != "dev_agent" {
		t.Fatalf("unexpected agent tag: %#v", got)
	}
}
