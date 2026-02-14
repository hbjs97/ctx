package testutil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Commander abstracts external command execution for testability.
type Commander interface {
	// Run executes an external command and returns its combined output.
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// RealCommander executes actual external commands.
type RealCommander struct{}

// Run executes the command using os/exec.
func (c *RealCommander) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// Response represents a pre-configured command response for FakeCommander.
type Response struct {
	Output []byte
	Err    error
}

// FakeCommander returns pre-configured responses for testing.
// Responses are keyed by "name arg1 arg2 ..." format.
// If no exact match is found, it tries prefix matching.
type FakeCommander struct {
	// Responses maps command strings to their responses.
	// Key format: "command arg1 arg2" (e.g., "git clone", "gh api repos/owner/repo")
	Responses map[string]Response

	// Calls records all commands that were executed, in order.
	Calls []string

	// DefaultResponse is returned when no matching response is found.
	// If nil, an error is returned for unmatched commands.
	DefaultResponse *Response
}

// NewFakeCommander creates a FakeCommander with an empty response map.
func NewFakeCommander() *FakeCommander {
	return &FakeCommander{
		Responses: make(map[string]Response),
	}
}

// Register adds a response for the given command key.
func (c *FakeCommander) Register(key string, output string, err error) {
	c.Responses[key] = Response{
		Output: []byte(output),
		Err:    err,
	}
}

// Run looks up the command in Responses and returns the matching response.
func (c *FakeCommander) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	fullCmd := name
	if len(args) > 0 {
		fullCmd = name + " " + strings.Join(args, " ")
	}

	c.Calls = append(c.Calls, fullCmd)

	// Exact match first.
	if resp, ok := c.Responses[fullCmd]; ok {
		return resp.Output, resp.Err
	}

	// Try prefix matching (longest prefix wins).
	bestKey := ""
	for key := range c.Responses {
		if strings.HasPrefix(fullCmd, key) && len(key) > len(bestKey) {
			bestKey = key
		}
	}
	if bestKey != "" {
		resp := c.Responses[bestKey]
		return resp.Output, resp.Err
	}

	// Default response.
	if c.DefaultResponse != nil {
		return c.DefaultResponse.Output, c.DefaultResponse.Err
	}

	return nil, fmt.Errorf("FakeCommander: no response registered for %q", fullCmd)
}

// Called returns true if a command matching the given prefix was executed.
func (c *FakeCommander) Called(prefix string) bool {
	for _, call := range c.Calls {
		if strings.HasPrefix(call, prefix) {
			return true
		}
	}
	return false
}

// CallCount returns the number of times a command matching the given prefix was executed.
func (c *FakeCommander) CallCount(prefix string) int {
	count := 0
	for _, call := range c.Calls {
		if strings.HasPrefix(call, prefix) {
			count++
		}
	}
	return count
}

// Reset clears all recorded calls.
func (c *FakeCommander) Reset() {
	c.Calls = nil
}
