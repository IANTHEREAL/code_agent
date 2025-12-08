package config

import (
	"testing"
	"time"
)

func TestFromEnvReadsStatusTimeout(t *testing.T) {
	t.Setenv("AZURE_OPENAI_API_KEY", "test-key")
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.com")
	t.Setenv("AZURE_OPENAI_DEPLOYMENT", "gpt4o")
	t.Setenv("AZURE_OPENAI_API_VERSION", "2024-05-01")
	t.Setenv("MCP_BASE_URL", "http://localhost:8000/mcp/sse")
	t.Setenv("PROJECT_NAME", "proj")
	t.Setenv("WORKSPACE_DIR", "/tmp/workspace")
	t.Setenv("GITHUB_TOKEN", "ghp_test")
	t.Setenv("GIT_AUTHOR_NAME", "Tester")
	t.Setenv("GIT_AUTHOR_EMAIL", "tester@example.com")
	t.Setenv("DEV_AGENT_STATUS_TIMEOUT_SECONDS", "900")

	conf, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv failed: %v", err)
	}
	if conf.StatusTimeout != 900*time.Second {
		t.Fatalf("expected status timeout 900s, got %s", conf.StatusTimeout)
	}
}
