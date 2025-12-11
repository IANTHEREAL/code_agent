package tools

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPClientAddsMetaToToolCalls(t *testing.T) {
	t.Parallel()

	type capturedRequest struct {
		body map[string]any
	}

	newClient := func(t *testing.T) (*MCPClient, *capturedRequest, func()) {
		t.Helper()
		req := &capturedRequest{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&req.body); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"result":{"structuredContent":{"ok":true}}}`)
		}))

		client := NewMCPClient(server.URL)
		client.client = server.Client()
		return client, req, func() { server.Close() }
	}

	assertMetaTag := func(t *testing.T, payload map[string]any) {
		t.Helper()
		params, ok := payload["params"].(map[string]any)
		if !ok {
			t.Fatalf("params missing or wrong type: %#v", payload["params"])
		}
		meta, ok := params["_meta"].(map[string]any)
		if !ok {
			t.Fatalf("_meta missing from params: %#v", params)
		}
		val, ok := meta["ai.tidb.pantheon-ai/agent"].(string)
		if !ok || val != "dev_agent" {
			t.Fatalf("unexpected meta tag value: %#v", meta)
		}
	}

	tests := []struct {
		name   string
		invoke func(t *testing.T, client *MCPClient) error
	}{
		{
			name: "CallTool",
			invoke: func(t *testing.T, client *MCPClient) error {
				_, err := client.CallTool("branch_output", map[string]any{"foo": "bar"})
				return err
			},
		},
		{
			name: "ParallelExplore",
			invoke: func(t *testing.T, client *MCPClient) error {
				_, err := client.ParallelExplore("proj", "parent", []string{"prompt"}, "codex", 2)
				return err
			},
		},
		{
			name: "GetBranch",
			invoke: func(t *testing.T, client *MCPClient) error {
				_, err := client.GetBranch("branch-123")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, req, cleanup := newClient(t)
			defer cleanup()

			if err := tt.invoke(t, client); err != nil {
				t.Fatalf("invoke %s failed: %v", tt.name, err)
			}
			if req.body == nil {
				t.Fatalf("no request captured for %s", tt.name)
			}
			assertMetaTag(t, req.body)
		})
	}
}
