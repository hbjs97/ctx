package resolver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hbjs97/ctx/internal/cache"
	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/gh"
	"github.com/hbjs97/ctx/internal/git"
)

// ErrAmbiguous는 복수 프로필이 매칭되어 자동 판정이 불가능할 때 반환된다.
var ErrAmbiguous = errors.New("모호한 판정, --profile 플래그 필요")

// ErrAuthFail는 접근 가능한 프로필이 없을 때 반환된다.
var ErrAuthFail = errors.New("접근 가능한 프로필 없음")

// Result는 Resolver의 판정 결과다.
type Result struct {
	Profile string
	Reason  string // "explicit", "cache", "owner_rule", "probe", "user_select"
}

// Resolver는 5단계 계정 판정 파이프라인이다.
type Resolver struct {
	config *config.Config
	cache  *cache.Cache
	git    *git.Adapter
	gh     *gh.Adapter
}

// New는 새 Resolver를 생성한다.
func New(cfg *config.Config, c *cache.Cache, g *git.Adapter, h *gh.Adapter) *Resolver {
	return &Resolver{config: cfg, cache: c, git: g, gh: h}
}

// Resolve는 5단계 파이프라인으로 프로필을 판정한다.
func (r *Resolver) Resolve(ctx context.Context, ownerRepo, explicitProfile string) (*Result, error) {
	// Step 1: 명시 플래그
	if explicitProfile != "" {
		if _, err := r.config.GetProfile(explicitProfile); err != nil {
			return nil, fmt.Errorf("resolver.Resolve: %w", err)
		}
		return &Result{Profile: explicitProfile, Reason: "explicit"}, nil
	}

	// Step 2: 캐시 조회
	configHash := r.config.ConfigHash()
	if entry, ok := r.cache.Lookup(ownerRepo, configHash, r.config.CacheTTLDays); ok {
		return &Result{Profile: entry.Profile, Reason: "cache"}, nil
	}

	// Step 3: Owner 규칙
	owner := strings.SplitN(ownerRepo, "/", 2)[0]
	matches := r.config.MatchOwner(owner)
	if len(matches) == 1 {
		return &Result{Profile: matches[0], Reason: "owner_rule"}, nil
	}

	// Step 4: 권한 Probe
	profileDirs := make(map[string]string)
	for name, p := range r.config.Profiles {
		profileDirs[name] = p.GHConfigDir
	}
	parts := strings.SplitN(ownerRepo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("resolver.Resolve: 잘못된 owner/repo: %s", ownerRepo)
	}
	probeResults, err := r.gh.ProbeAllProfiles(ctx, parts[0], parts[1], profileDirs)
	if err != nil {
		return nil, fmt.Errorf("resolver.Resolve: %w", err)
	}

	var pushable []string
	for _, pr := range probeResults {
		if pr.CanPush {
			pushable = append(pushable, pr.Profile)
		}
	}

	if len(pushable) == 1 {
		return &Result{Profile: pushable[0], Reason: "probe"}, nil
	}
	if len(pushable) == 0 {
		return nil, fmt.Errorf("resolver.Resolve: %w", ErrAuthFail)
	}

	// Step 5: 복수 프로필 매칭 — 자동 판정 불가
	return nil, fmt.Errorf("resolver.Resolve: %w", ErrAmbiguous)
}
