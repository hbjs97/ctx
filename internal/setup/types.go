package setup

// Action은 재실행 시 사용자가 선택하는 작업이다.
type Action string

const (
	ActionAdd    Action = "add"
	ActionEdit   Action = "edit"
	ActionDelete Action = "delete"
)

// ProfileInput은 프로필 생성/수정 시 사용자 입력 값이다.
type ProfileInput struct {
	Name     string
	GitName  string
	GitEmail string
	SSHHost  string
	Owners   []string
}

// SSHKeyInfo는 ~/.ssh 디렉토리에서 발견된 SSH 키 쌍 정보다.
type SSHKeyInfo struct {
	// Name은 키 파일 이름이다 (예: "id_ed25519_work").
	Name string
	// PrivateKey는 개인 키의 전체 경로다.
	PrivateKey string
	// PublicKey는 공개 키(.pub)의 전체 경로다.
	PublicKey string
}

// SSHKeyChoice는 SSH 키 선택 결과를 나타낸다.
type SSHKeyChoice struct {
	// Action은 사용자 선택 액션이다 ("existing" | "generate" | "skip").
	Action string
	// ExistingKey는 Action이 "existing"일 때 선택된 개인 키 경로다.
	ExistingKey string
}

// FormRunner는 TUI 폼 실행을 추상화하는 interface다.
// 프로덕션에서는 huh 기반 구현, 테스트에서는 mock을 사용한다.
type FormRunner interface {
	// RunProfileForm은 프로필 입력 폼을 실행한다.
	// defaults가 nil이 아니면 기존 값을 기본값으로 표시한다 (수정 모드).
	RunProfileForm(defaults *ProfileInput, existingNames []string) (*ProfileInput, error)

	// RunActionSelect는 작업 선택 UI를 표시한다.
	RunActionSelect(profileNames []string) (Action, error)

	// RunProfileSelect는 프로필 선택 UI를 표시한다.
	RunProfileSelect(profileNames []string) (string, error)

	// RunConfirm은 확인 프롬프트를 표시한다.
	RunConfirm(message string) (bool, error)

	// RunAddMore는 "프로필을 더 추가하시겠습니까?" 프롬프트를 표시한다.
	RunAddMore() (bool, error)

	// RunSSHHostSelect는 감지된 SSH host 목록에서 선택 UI를 표시한다.
	// hosts가 비어있으면 직접 입력 폼으로 fallback한다.
	RunSSHHostSelect(hosts []string) (string, error)

	// RunOwnersSelect는 조직/사용자 체크리스트를 표시한다.
	// detected가 비어있으면 직접 입력 폼으로 fallback한다.
	RunOwnersSelect(detected []string) ([]string, error)

	// RunSSHKeySelect는 SSH 키 선택 UI를 표시한다.
	// 기존 키 목록에서 선택하거나, 새로 생성하거나, 건너뛸 수 있다.
	RunSSHKeySelect(existingKeys []SSHKeyInfo, profileName string) (SSHKeyChoice, error)
}
