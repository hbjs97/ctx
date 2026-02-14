package shell

import (
	"fmt"

	"github.com/hbjs97/ctx/internal/config"
)

// Activate는 프로필 활성화를 위한 shell export 명령을 생성한다.
func Activate(profileName string, profile *config.Profile, shellType string) string {
	switch shellType {
	case "fish":
		return fmt.Sprintf(
			"set -gx GH_CONFIG_DIR %q\nset -gx CTX_PROFILE %q\n",
			profile.GHConfigDir, profileName,
		)
	default: // bash, zsh, sh
		return fmt.Sprintf(
			"export GH_CONFIG_DIR=%q\nexport CTX_PROFILE=%q\n",
			profile.GHConfigDir, profileName,
		)
	}
}

// Deactivate는 프로필 비활성화를 위한 shell unset 명령을 생성한다.
func Deactivate(shellType string) string {
	switch shellType {
	case "fish":
		return "set -e GH_CONFIG_DIR\nset -e CTX_PROFILE\n"
	default:
		return "unset GH_CONFIG_DIR\nunset CTX_PROFILE\n"
	}
}

// HookSnippet는 셸 디렉토리 변경 hook 스니펫을 반환한다.
func HookSnippet(shellType string) string {
	switch shellType {
	case "zsh":
		return `# ctx shell integration (zsh)
_ctx_chpwd() {
  eval "$(ctx activate --shell zsh 2>/dev/null)"
}
chpwd_functions+=(_ctx_chpwd)
`
	case "bash":
		return `# ctx shell integration (bash)
_ctx_prompt_command() {
  eval "$(ctx activate --shell bash 2>/dev/null)"
}
PROMPT_COMMAND="_ctx_prompt_command;${PROMPT_COMMAND}"
`
	case "fish":
		return `# ctx shell integration (fish)
function _ctx_chpwd --on-variable PWD
  eval (ctx activate --shell fish 2>/dev/null)
end
`
	default:
		return ""
	}
}
