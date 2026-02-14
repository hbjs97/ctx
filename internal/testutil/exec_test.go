package testutil

import (
	"context"
	"fmt"
	"testing"
)

func TestFakeCommander_ExactMatch(t *testing.T) {
	t.Parallel()

	fc := NewFakeCommander()
	fc.Register("git status", "clean\n", nil)

	out, err := fc.Run(context.Background(), "git", "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "clean\n" {
		t.Errorf("got %q, want %q", string(out), "clean\n")
	}
}

func TestFakeCommander_PrefixMatch(t *testing.T) {
	t.Parallel()

	fc := NewFakeCommander()
	fc.Register("gh api", `{"permissions":{"push":true}}`, nil)

	out, err := fc.Run(context.Background(), "gh", "api", "repos/owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != `{"permissions":{"push":true}}` {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestFakeCommander_NoMatch(t *testing.T) {
	t.Parallel()

	fc := NewFakeCommander()

	_, err := fc.Run(context.Background(), "unknown", "command")
	if err == nil {
		t.Fatal("expected error for unregistered command")
	}
}

func TestFakeCommander_DefaultResponse(t *testing.T) {
	t.Parallel()

	fc := NewFakeCommander()
	fc.DefaultResponse = &Response{Output: []byte("default"), Err: nil}

	out, err := fc.Run(context.Background(), "any", "command")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "default" {
		t.Errorf("got %q, want %q", string(out), "default")
	}
}

func TestFakeCommander_RecordsCalls(t *testing.T) {
	t.Parallel()

	fc := NewFakeCommander()
	fc.DefaultResponse = &Response{Output: nil, Err: nil}

	fc.Run(context.Background(), "git", "status")
	fc.Run(context.Background(), "gh", "api", "repos/o/r")

	if len(fc.Calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(fc.Calls))
	}
	if !fc.Called("git") {
		t.Error("expected git to be called")
	}
	if fc.CallCount("gh") != 1 {
		t.Errorf("expected 1 gh call, got %d", fc.CallCount("gh"))
	}
}

func TestFakeCommander_ErrorResponse(t *testing.T) {
	t.Parallel()

	fc := NewFakeCommander()
	fc.Register("git push", "error: failed\n", fmt.Errorf("exit status 1"))

	out, err := fc.Run(context.Background(), "git", "push")
	if err == nil {
		t.Fatal("expected error")
	}
	if string(out) != "error: failed\n" {
		t.Errorf("got %q, want %q", string(out), "error: failed\n")
	}
}
