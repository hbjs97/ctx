package setup

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
)

// HuhFormRunner는 charmbracelet/huh 기반의 FormRunner 구현이다.
type HuhFormRunner struct{}

var _ FormRunner = (*HuhFormRunner)(nil)

var profileNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`)

// RunProfileForm은 프로필 입력 폼을 실행한다.
func (h *HuhFormRunner) RunProfileForm(defaults *ProfileInput, existingNames []string) (*ProfileInput, error) {
	input := &ProfileInput{}
	if defaults != nil {
		*input = *defaults
	}

	nameValidate := func(s string) error {
		if s == "" {
			return fmt.Errorf("프로필 이름을 입력하세요")
		}
		if !profileNameRegex.MatchString(s) {
			return fmt.Errorf("영문, 숫자, 하이픈만 사용 가능합니다")
		}
		for _, n := range existingNames {
			if n == s && (defaults == nil || defaults.Name != s) {
				return fmt.Errorf("이미 존재하는 프로필 이름입니다: %s", s)
			}
		}
		return nil
	}

	emailValidate := func(s string) error {
		if !strings.Contains(s, "@") {
			return fmt.Errorf("올바른 이메일 형식이 아닙니다")
		}
		return nil
	}

	fields := []huh.Field{
		huh.NewInput().Title("프로필 이름").Value(&input.Name).Validate(nameValidate),
		huh.NewInput().Title("git user.name").Value(&input.GitName).Validate(huh.ValidateNotEmpty()),
		huh.NewInput().Title("git user.email").Value(&input.GitEmail).Validate(emailValidate),
	}

	form := huh.NewForm(huh.NewGroup(fields...))
	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("setup.RunProfileForm: %w", err)
	}

	return input, nil
}

// RunActionSelect는 작업 선택 UI를 표시한다.
func (h *HuhFormRunner) RunActionSelect(profileNames []string) (Action, error) {
	var action Action
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[Action]().
			Title("작업을 선택하세요").
			Options(
				huh.NewOption("프로필 추가", ActionAdd),
				huh.NewOption("프로필 수정", ActionEdit),
				huh.NewOption("프로필 삭제", ActionDelete),
			).
			Value(&action),
	))
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("setup.RunActionSelect: %w", err)
	}
	return action, nil
}

// RunProfileSelect는 프로필 선택 UI를 표시한다.
func (h *HuhFormRunner) RunProfileSelect(profileNames []string) (string, error) {
	var selected string
	options := make([]huh.Option[string], len(profileNames))
	for i, name := range profileNames {
		options[i] = huh.NewOption(name, name)
	}

	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("프로필을 선택하세요").
			Options(options...).
			Value(&selected),
	))
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("setup.RunProfileSelect: %w", err)
	}
	return selected, nil
}

// RunConfirm은 확인 프롬프트를 표시한다.
func (h *HuhFormRunner) RunConfirm(message string) (bool, error) {
	var confirm bool
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(message).Value(&confirm),
	))
	if err := form.Run(); err != nil {
		return false, fmt.Errorf("setup.RunConfirm: %w", err)
	}
	return confirm, nil
}

// RunAddMore는 "프로필을 더 추가하시겠습니까?" 프롬프트를 표시한다.
func (h *HuhFormRunner) RunAddMore() (bool, error) {
	return h.RunConfirm("프로필을 더 추가하시겠습니까?")
}

// RunSSHHostSelect는 SSH host 선택 UI를 표시한다.
func (h *HuhFormRunner) RunSSHHostSelect(hosts []string) (string, error) {
	if len(hosts) == 0 {
		var host string
		form := huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("SSH host alias").
				Description("~/.ssh/config의 Host 값 (예: github.com-work)").
				Value(&host).
				Validate(huh.ValidateNotEmpty()),
		))
		if err := form.Run(); err != nil {
			return "", fmt.Errorf("setup.RunSSHHostSelect: %w", err)
		}
		return host, nil
	}

	var selected string
	options := make([]huh.Option[string], 0, len(hosts)+1)
	for _, h := range hosts {
		options = append(options, huh.NewOption(h, h))
	}
	options = append(options, huh.NewOption("직접 입력...", "__manual__"))

	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("SSH host를 선택하세요").
			Options(options...).
			Value(&selected),
	))
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("setup.RunSSHHostSelect: %w", err)
	}

	if selected == "__manual__" {
		return h.RunSSHHostSelect(nil)
	}
	return selected, nil
}

// RunSSHKeySelect는 SSH 키 선택 UI를 표시한다.
func (h *HuhFormRunner) RunSSHKeySelect(existingKeys []SSHKeyInfo, profileName string) (SSHKeyChoice, error) {
	if len(existingKeys) == 0 {
		var generate bool
		form := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().
				Title("SSH 키가 없습니다. 새로 생성할까요?").
				Description(fmt.Sprintf("~/.ssh/id_ed25519_%s 키 쌍을 생성합니다", profileName)).
				Value(&generate),
		))
		if err := form.Run(); err != nil {
			return SSHKeyChoice{}, fmt.Errorf("setup.RunSSHKeySelect: %w", err)
		}
		if generate {
			return SSHKeyChoice{Action: "generate"}, nil
		}
		return SSHKeyChoice{Action: "skip"}, nil
	}

	options := make([]huh.Option[string], 0, len(existingKeys)+1)
	for _, k := range existingKeys {
		options = append(options, huh.NewOption(k.Name+" ("+k.PublicKey+")", k.PrivateKey))
	}
	options = append(options, huh.NewOption("새 키 생성 (id_ed25519_"+profileName+")", "__generate__"))

	var selected string
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("SSH 키를 선택하세요").
			Options(options...).
			Value(&selected),
	))
	if err := form.Run(); err != nil {
		return SSHKeyChoice{}, fmt.Errorf("setup.RunSSHKeySelect: %w", err)
	}

	if selected == "__generate__" {
		return SSHKeyChoice{Action: "generate"}, nil
	}
	return SSHKeyChoice{Action: "existing", ExistingKey: selected}, nil
}

// RunOwnersSelect는 owners 선택 UI를 표시한다.
func (h *HuhFormRunner) RunOwnersSelect(detected []string) ([]string, error) {
	if len(detected) == 0 {
		var input string
		form := huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("owners (콤마 구분)").
				Description("GitHub 조직명 또는 사용자명").
				Value(&input).
				Validate(huh.ValidateNotEmpty()),
		))
		if err := form.Run(); err != nil {
			return nil, fmt.Errorf("setup.RunOwnersSelect: %w", err)
		}
		var owners []string
		for _, o := range strings.Split(input, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				owners = append(owners, o)
			}
		}
		return owners, nil
	}

	selected := make([]string, len(detected))
	copy(selected, detected)

	form := huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("이 계정으로 접근 가능한 조직/사용자").
			Options(func() []huh.Option[string] {
				opts := make([]huh.Option[string], len(detected))
				for i, d := range detected {
					opts[i] = huh.NewOption(d, d).Selected(true)
				}
				return opts
			}()...).
			Value(&selected),
	))
	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("setup.RunOwnersSelect: %w", err)
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("최소 1개 이상 선택해야 합니다")
	}
	return selected, nil
}
