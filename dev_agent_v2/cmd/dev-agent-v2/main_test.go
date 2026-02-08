package main

import "testing"

func TestSanitizeFinalReportRemovesRedundantFieldsAndNulls(t *testing.T) {
	report := map[string]any{
		"is_finished":       true,
		"status":            "completed",
		"summary":           "ok",
		"task":              "fix something",
		"pr_url":            nil,
		"pr_number":         nil,
		"pr_head_branch":    nil,
		"instructions":      "",
		"start_branch_id":   "start",
		"latest_branch_id":  "latest",
		"some_other_field":  "drop",
		"some_other_number": 123,
	}

	sanitizeFinalReport(report)

	if _, ok := report["pr_url"]; ok {
		t.Fatalf("expected pr_url to be removed when nil")
	}
	if _, ok := report["pr_number"]; ok {
		t.Fatalf("expected pr_number to be removed when nil")
	}
	if _, ok := report["pr_head_branch"]; ok {
		t.Fatalf("expected pr_head_branch to be removed when nil")
	}
	if _, ok := report["instructions"]; ok {
		t.Fatalf("expected instructions to be removed when empty")
	}
	if got, ok := report["task"]; !ok || got == "" {
		t.Fatalf("expected task to be preserved, got %#v", got)
	}
	if _, ok := report["some_other_field"]; ok {
		t.Fatalf("expected unknown fields to be removed")
	}
	if _, ok := report["some_other_number"]; ok {
		t.Fatalf("expected unknown fields to be removed")
	}
}

func TestSanitizeFinalReportKeepsPRFieldsWhenPresent(t *testing.T) {
	report := map[string]any{
		"task":           "fix",
		"pr_number":      float64(123),
		"pr_url":         "https://example.com/pr/123",
		"pr_head_branch": "feature-branch",
	}

	sanitizeFinalReport(report)

	if report["pr_number"] == nil {
		t.Fatalf("expected pr_number to be kept")
	}
	if report["pr_url"] != "https://example.com/pr/123" {
		t.Fatalf("unexpected pr_url %#v", report["pr_url"])
	}
	if report["pr_head_branch"] != "feature-branch" {
		t.Fatalf("unexpected pr_head_branch %#v", report["pr_head_branch"])
	}
}
