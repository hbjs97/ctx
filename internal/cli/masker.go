package cli

import "regexp"

var tokenPattern = regexp.MustCompile(`(ghp_|gho_|github_pat_|ghs_|ghu_)\S+`)

// MaskTokens는 GitHub 토큰 패턴을 마스킹한다.
func MaskTokens(s string) string {
	return tokenPattern.ReplaceAllStringFunc(s, func(match string) string {
		for _, prefix := range []string{"ghp_", "gho_", "github_pat_", "ghs_", "ghu_"} {
			if len(match) >= len(prefix) && match[:len(prefix)] == prefix {
				return prefix + "****"
			}
		}
		return match
	})
}
