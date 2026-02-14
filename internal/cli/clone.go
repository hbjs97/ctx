package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hbjs97/ctx/internal/cache"
	"github.com/hbjs97/ctx/internal/cmdexec"
	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/gh"
	"github.com/hbjs97/ctx/internal/git"
	"github.com/hbjs97/ctx/internal/guard"
	"github.com/hbjs97/ctx/internal/resolver"
	"github.com/spf13/cobra"
)

func newCloneCmd() *cobra.Command {
	var profileFlag string
	var noGuard bool

	cmd := &cobra.Command{
		Use:   "clone <repo>",
		Short: "리포를 클론하고 프로필을 자동 설정한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClone(cmd.Context(), args[0], profileFlag, noGuard)
		},
	}
	cmd.Flags().StringVarP(&profileFlag, "profile", "p", "", "사용할 프로필 이름")
	cmd.Flags().BoolVar(&noGuard, "no-guard", false, "pre-push guard 설치 생략")
	return cmd
}

func runClone(ctx context.Context, target, profileFlag string, noGuard bool) error {
	ref, err := git.ParseRepoURL(target)
	if err != nil {
		return err
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	c, _ := cache.Load(cachePath()) // 캐시 로드 실패 시 빈 캐시 사용

	commander := &cmdexec.RealCommander{}
	gitAdapter := git.NewAdapter(commander)
	ghAdapter := gh.NewAdapter(commander)

	ownerRepo := ref.Owner + "/" + ref.Repo
	r := resolver.New(cfg, c, gitAdapter, ghAdapter, false)
	result, err := r.Resolve(ctx, ownerRepo, profileFlag)
	if err != nil {
		return err
	}

	profile, _ := cfg.GetProfile(result.Profile) // Resolve 성공이면 프로필 존재 보장
	remoteURL := git.BuildSSHRemoteURL(profile.SSHHost, ref.Owner, ref.Repo)
	destDir := ref.Repo

	if err := gitAdapter.Clone(ctx, remoteURL, destDir); err != nil {
		return err
	}

	absDir, _ := filepath.Abs(destDir) // clone 직후이므로 실패 가능성 낮음
	_ = gitAdapter.SetLocalConfig(ctx, absDir, "user.name", profile.GitName)   // clone 직후이므로 에러 무시
	_ = gitAdapter.SetLocalConfig(ctx, absDir, "user.email", profile.GitEmail) // clone 직후이므로 에러 무시

	profilePath := filepath.Join(absDir, ".git", "ctx-profile")
	_ = os.WriteFile(profilePath, []byte(result.Profile+"\n"), 0600) // clone 직후이므로 실패 가능성 낮음

	if !noGuard && cfg.IsRequirePushGuard() {
		_ = guard.InstallHook(absDir) // guard 설치 실패는 치명적이지 않음
	}

	c.Set(ownerRepo, cache.Entry{
		Profile:    result.Profile,
		Reason:     result.Reason,
		ResolvedAt: time.Now().Format(time.RFC3339),
		ConfigHash: cfg.ConfigHash(),
	})
	_ = c.Save(cachePath()) // 캐시 저장 실패는 치명적이지 않음

	fmt.Printf("클론 완료: %s → 프로필: %s (판정: %s)\n", ownerRepo, result.Profile, result.Reason)
	return nil
}
