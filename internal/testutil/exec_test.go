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

func TestFakeCommander_RunWithEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		env        map[string]string
		cmd        string
		args       []string
		register   string
		output     string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "delegates to Run logic with env recorded",
			env:        map[string]string{"GH_CONFIG_DIR": "/home/user/.config/gh-work"},
			cmd:        "gh",
			args:       []string{"api", "repos/owner/repo"},
			register:   "gh api",
			output:     `{"permissions":{"push":true}}`,
			wantOutput: `{"permissions":{"push":true}}`,
		},
		{
			name:       "records env with nil map",
			env:        nil,
			cmd:        "git",
			args:       []string{"status"},
			register:   "git status",
			output:     "clean\n",
			wantOutput: "clean\n",
		},
		{
			name:       "records env with empty map",
			env:        map[string]string{},
			cmd:        "git",
			args:       []string{"log"},
			register:   "git log",
			output:     "commit abc\n",
			wantOutput: "commit abc\n",
		},
		{
			name:       "records multiple env vars",
			env:        map[string]string{"GH_CONFIG_DIR": "/tmp/gh", "GH_TOKEN": "ghp_test"},
			cmd:        "gh",
			args:       []string{"auth", "status"},
			register:   "gh auth status",
			output:     "Logged in",
			wantOutput: "Logged in",
		},
		{
			name:    "returns error for unregistered command",
			env:     map[string]string{"FOO": "bar"},
			cmd:     "unknown",
			args:    []string{"cmd"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fc := NewFakeCommander()
			if tt.register != "" {
				fc.Register(tt.register, tt.output, nil)
			}

			out, err := fc.RunWithEnv(context.Background(), tt.env, tt.cmd, tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(out) != tt.wantOutput {
				t.Errorf("output: got %q, want %q", string(out), tt.wantOutput)
			}
		})
	}
}

func TestFakeCommander_RunWithEnv_RecordsEnvCalls(t *testing.T) {
	t.Parallel()

	fc := NewFakeCommander()
	fc.DefaultResponse = &Response{Output: nil, Err: nil}

	env1 := map[string]string{"GH_CONFIG_DIR": "/path/one"}
	env2 := map[string]string{"GH_CONFIG_DIR": "/path/two", "EXTRA": "val"}

	fc.RunWithEnv(context.Background(), env1, "gh", "api", "repos/a/b")
	fc.RunWithEnv(context.Background(), env2, "gh", "auth", "status")

	// EnvCalls should record both env maps.
	if len(fc.EnvCalls) != 2 {
		t.Fatalf("expected 2 EnvCalls, got %d", len(fc.EnvCalls))
	}
	if fc.EnvCalls[0]["GH_CONFIG_DIR"] != "/path/one" {
		t.Errorf("EnvCalls[0] GH_CONFIG_DIR: got %q, want %q", fc.EnvCalls[0]["GH_CONFIG_DIR"], "/path/one")
	}
	if fc.EnvCalls[1]["GH_CONFIG_DIR"] != "/path/two" {
		t.Errorf("EnvCalls[1] GH_CONFIG_DIR: got %q, want %q", fc.EnvCalls[1]["GH_CONFIG_DIR"], "/path/two")
	}
	if fc.EnvCalls[1]["EXTRA"] != "val" {
		t.Errorf("EnvCalls[1] EXTRA: got %q, want %q", fc.EnvCalls[1]["EXTRA"], "val")
	}

	// Calls should also be recorded (delegates to Run logic).
	if len(fc.Calls) != 2 {
		t.Fatalf("expected 2 Calls, got %d", len(fc.Calls))
	}
	if !fc.Called("gh api") {
		t.Error("expected 'gh api' to be called")
	}
	if !fc.Called("gh auth") {
		t.Error("expected 'gh auth' to be called")
	}
}

func TestFakeCommander_RunWithEnv_NilEnvRecordsNil(t *testing.T) {
	t.Parallel()

	fc := NewFakeCommander()
	fc.DefaultResponse = &Response{Output: nil, Err: nil}

	fc.RunWithEnv(context.Background(), nil, "git", "status")

	if len(fc.EnvCalls) != 1 {
		t.Fatalf("expected 1 EnvCalls, got %d", len(fc.EnvCalls))
	}
	if fc.EnvCalls[0] != nil {
		t.Errorf("expected nil EnvCalls[0], got %v", fc.EnvCalls[0])
	}
}
