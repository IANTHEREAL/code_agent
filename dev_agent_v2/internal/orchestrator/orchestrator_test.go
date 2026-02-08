package orchestrator

import (
	"testing"
)

func TestToolInstructionExtractsInstruction(t *testing.T) {
	resp := map[string]any{
		"status": "error",
		"error": map[string]any{
			"message":     "code review log was missing",
			"instruction": "FINISHED_WITH_ERROR",
			"details":     map[string]any{"attempts": 3},
		},
	}

	instr, msg, payload := toolInstruction(resp)
	if instr != "FINISHED_WITH_ERROR" {
		t.Fatalf("expected FINISHED_WITH_ERROR, got %q", instr)
	}
	if msg != "code review log was missing" {
		t.Fatalf("expected message propagation, got %q", msg)
	}
	got, ok := payload["attempts"]
	if !ok {
		t.Fatalf("expected attempts detail to exist")
	}
	switch v := got.(type) {
	case int:
		if v != 3 {
			t.Fatalf("expected attempts=3, got %d", v)
		}
	case float64:
		if int(v) != 3 {
			t.Fatalf("expected attempts=3, got %f", v)
		}
	default:
		t.Fatalf("unexpected attempts type %T", got)
	}
}

func TestToolInstructionHandlesStringErrors(t *testing.T) {
	resp := map[string]any{
		"status": "error",
		"error":  "something failed",
	}
	instr, msg, payload := toolInstruction(resp)
	if instr != "" {
		t.Fatalf("expected no instruction, got %q", instr)
	}
	if msg != "something failed" {
		t.Fatalf("expected message fallback, got %q", msg)
	}
	if payload != nil {
		t.Fatalf("expected nil payload, got %#v", payload)
	}
}
