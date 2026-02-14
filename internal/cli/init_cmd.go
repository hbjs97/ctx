package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hbjs97/ctx/internal/cache"
	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/gh"
	"github.com/hbjs97/ctx/internal/git"
	"github.com/hbjs97/ctx/internal/guard"
	"github.com/hbjs97/ctx/internal/resolver"
	"github.com/spf13/cobra"
)

func (a *App) newInitCmd() *cobra.Command {
	var profileFlag string
	var noGuard bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "현재 리포에 ctx 프로필을 설정한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runInit(cmd.Context(), profileFlag, noGuard)
		},
	}
	cmd.Flags().StringVarP(&profileFlag, "profile", "p", "", "사용할 프로필 이름")
	cmd.Flags().BoolVar(&noGuard, "no-guard", false, "pre-push guard 설치 생략")
	return cmd
}

func (a *App) runInit(ctx context.Context, profileFlag string, noGuard bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cli.init: %w", err)
	}

	cfg, err := config.Load(a.CfgPath)
	if err != nil {
		return err
	}

	gitAdapter := git.NewAdapter(a.Commander)
	ghAdapter := gh.NewAdapter(a.Commander)

	remoteURL, err := gitAdapter.GetRemoteURL(ctx, cwd, "origin")
	if err != nil {
		return fmt.Errorf("cli.init: origin remote 없음: %w", err)
	}

	ref, err := git.ParseRepoURL(remoteURL)
	if err != nil {
		return err
	}

	c, _ := cache.Load(a.cachePath()) // 캐시 로드 실패 시 빈 캐시 사용
	ownerRepo := ref.Owner + "/" + ref.Repo

	r := resolver.New(cfg, c, gitAdapter, ghAdapter, false)
	result, err := r.Resolve(ctx, ownerRepo, profileFlag)
	if err != nil {
		return err
	}

	profile, _ := cfg.GetProfile(result.Profile) // Resolve 성공이면 프로필 존재 보장

	// HTTPS -> SSH 변환
	if git.IsHTTPSRemote(remoteURL) && !cfg.AllowHTTPSManagedRepo {
		newURL := git.BuildSSHRemoteURL(profile.SSHHost, ref.Owner, ref.Repo)
		if err := gitAdapter.SetRemoteURL(ctx, cwd, "origin", newURL); err != nil {
			return fmt.Errorf("cli.init: remote URL 변경 실패: %w", err)
		}
		fmt.Printf("remote URL 변경: %s → %s\n", remoteURL, newURL)
	}

	_ = gitAdapter.SetLocalConfig(ctx, cwd, "user.name", profile.GitName)   // 설정 실패는 치명적이지 않음
	_ = gitAdapter.SetLocalConfig(ctx, cwd, "user.email", profile.GitEmail) // 설정 실패는 치명적이지 않음

	profilePath := filepath.Join(cwd, ".git", "ctx-profile")
	_ = os.WriteFile(profilePath, []byte(result.Profile+"\n"), 0600) // .git 존재 확인 후이므로 실패 가능성 낮음

	if !noGuard && cfg.IsRequirePushGuard() {
		_ = guard.InstallHook(cwd) // guard 설치 실패는 치명적이지 않음
	}

	c.Set(ownerRepo, cache.Entry{
		Profile:    result.Profile,
		Reason:     result.Reason,
		ResolvedAt: time.Now().Format(time.RFC3339),
		ConfigHash: cfg.ConfigHash(),
	})
	_ = c.Save(a.cachePath()) // 캐시 저장 실패는 치명적이지 않음

	fmt.Printf("초기화 완료: %s → 프로필: %s (판정: %s)\n", ownerRepo, result.Profile, result.Reason)
	return nil
}
