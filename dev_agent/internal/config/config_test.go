package config

import (
	"testing"
	"time"
)

func mustSetBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AZURE_OPENAI_API_KEY", "key")
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.net")
	t.Setenv("AZURE_OPENAI_DEPLOYMENT", "dep")
	t.Setenv("AZURE_OPENAI_API_VERSION", "2024-12-01-preview")
	t.Setenv("MCP_BASE_URL", "http://localhost:8000/mcp/sse")
	t.Setenv("GITHUB_TOKEN", "ghp_test")
	t.Setenv("GIT_AUTHOR_NAME", "Dev Agent")
	t.Setenv("GIT_AUTHOR_EMAIL", "dev@example.com")
	t.Setenv("PROJECT_NAME", "proj")
	t.Setenv("WORKSPACE_DIR", "/tmp")
}

func TestFromEnvSetsDefaultBranchTimeout(t *testing.T) {
	mustSetBaseEnv(t)
	t.Setenv("BRANCH_STATUS_TIMEOUT_SECONDS", "")

	conf, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv returned error: %v", err)
	}
	if got, want := conf.BranchStatusTimeout, 30*time.Minute; got != want {
		t.Fatalf("expected default BranchStatusTimeout=%s, got %s", want, got)
	}
}

func TestFromEnvHonorsBranchTimeoutEnv(t *testing.T) {
	mustSetBaseEnv(t)
	t.Setenv("BRANCH_STATUS_TIMEOUT_SECONDS", "7200")

	conf, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv returned error: %v", err)
	}
	if got, want := conf.BranchStatusTimeout, 2*time.Hour; got != want {
		t.Fatalf("expected BranchStatusTimeout=%s, got %s", want, got)
	}
}
