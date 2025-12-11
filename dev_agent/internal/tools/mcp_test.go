package tools

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPClientAddsMetaToToolCalls(t *testing.T) {
	requests := make(chan map[string]any, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading request body: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed unmarshaling request body: %v\nBody: %s", err, string(body))
		}
		requests <- payload
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":{"branch_id":"branch-xyz","content":"ok"}}`))
	}))
	defer server.Close()

	client := NewMCPClient(server.URL)
	client.client = server.Client()

	testCases := []struct {
		name   string
		invoke func(t *testing.T, c *MCPClient)
	}{
		{
			name: "ParallelExplore",
			invoke: func(t *testing.T, c *MCPClient) {
				_, err := c.ParallelExplore("proj", "parent", []string{"prompt"}, "agent", 1)
				if err != nil {
					t.Fatalf("ParallelExplore error: %v", err)
				}
			},
		},
		{
			name: "GetBranch",
			invoke: func(t *testing.T, c *MCPClient) {
				_, err := c.GetBranch("branch-xyz")
				if err != nil {
					t.Fatalf("GetBranch error: %v", err)
				}
			},
		},
		{
			name: "BranchReadFile",
			invoke: func(t *testing.T, c *MCPClient) {
				_, err := c.BranchReadFile("branch-xyz", "/file.txt")
				if err != nil {
					t.Fatalf("BranchReadFile error: %v", err)
				}
			},
		},
		{
			name: "BranchOutput",
			invoke: func(t *testing.T, c *MCPClient) {
				_, err := c.BranchOutput("branch-xyz", true)
				if err != nil {
					t.Fatalf("BranchOutput error: %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.invoke(t, client)
			select {
			case payload := <-requests:
				assertMetaTag(t, payload)
			default:
				t.Fatalf("no request captured for %s", tc.name)
			}
		})
	}
}

func assertMetaTag(t *testing.T, payload map[string]any) {
	t.Helper()
	params, ok := payload["params"].(map[string]any)
	if !ok {
		t.Fatalf("params missing or wrong type in payload: %#v", payload["params"])
	}
	meta, ok := params["_meta"].(map[string]any)
	if !ok {
		t.Fatalf("_meta missing or wrong type in params: %#v", params["_meta"])
	}
	got := meta["ai.tidb.pantheon-ai/agent"]
	if got != "dev_agent" {
		t.Fatalf("unexpected _meta value: %#v", got)
	}
}
