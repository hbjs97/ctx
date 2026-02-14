package testutil

import (
	"context"
	"fmt"
	"strings"
)

// Response는 FakeCommander의 사전 설정된 명령 응답이다.
type Response struct {
	Output []byte
	Err    error
}

// FakeCommander는 테스트용으로 사전 설정된 응답을 반환한다.
// 응답은 "name arg1 arg2 ..." 형식의 키로 매핑된다.
// 정확한 매칭이 없으면 prefix 매칭을 시도한다.
type FakeCommander struct {
	// Responses는 명령 문자열을 응답에 매핑한다.
	// 키 형식: "command arg1 arg2" (예: "git clone", "gh api repos/owner/repo")
	Responses map[string]Response

	// Calls는 실행된 모든 명령을 순서대로 기록한다.
	Calls []string

	// EnvCalls는 RunWithEnv에 전달된 환경변수 맵을 순서대로 기록한다.
	EnvCalls []map[string]string

	// DefaultResponse는 매칭되는 응답이 없을 때 반환된다.
	// nil이면 미매칭 명령에 대해 에러를 반환한다.
	DefaultResponse *Response
}

// NewFakeCommander는 빈 응답 맵으로 FakeCommander를 생성한다.
func NewFakeCommander() *FakeCommander {
	return &FakeCommander{
		Responses: make(map[string]Response),
	}
}

// Register는 주어진 명령 키에 대한 응답을 등록한다.
func (c *FakeCommander) Register(key string, output string, err error) {
	c.Responses[key] = Response{
		Output: []byte(output),
		Err:    err,
	}
}

// Run은 Responses에서 명령을 조회하여 매칭되는 응답을 반환한다.
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

// RunWithEnv는 환경변수를 기록하고 Run 로직에 위임한다.
func (c *FakeCommander) RunWithEnv(ctx context.Context, env map[string]string, name string, args ...string) ([]byte, error) {
	c.EnvCalls = append(c.EnvCalls, env)
	return c.Run(ctx, name, args...)
}

// Called는 주어진 prefix와 매칭되는 명령이 실행되었으면 true를 반환한다.
func (c *FakeCommander) Called(prefix string) bool {
	for _, call := range c.Calls {
		if strings.HasPrefix(call, prefix) {
			return true
		}
	}
	return false
}

// CallCount는 주어진 prefix와 매칭되는 명령이 실행된 횟수를 반환한다.
func (c *FakeCommander) CallCount(prefix string) int {
	count := 0
	for _, call := range c.Calls {
		if strings.HasPrefix(call, prefix) {
			count++
		}
	}
	return count
}
