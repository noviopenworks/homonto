package tocli

import (
	"strings"
	"testing"
)

func TestPlanContractFindings(t *testing.T) {
	t.Run("complete checked and unchecked tasks", func(t *testing.T) {
		plan := `# Goal
- [x] first outcome
  - Files: internal/first.go
  - Change: preserve the first contract
  - Verify: go test ./internal/tocli
- [ ] second outcome
  - Files: internal/second.go
  - Change: add the second contract
  - Verify: go test ./internal/tocli
Final Verify: go test ./...
`
		if findings := planContractFindings(plan); len(findings) != 0 {
			t.Fatalf("planContractFindings() = %v, want no findings", findings)
		}
	})

	t.Run("reports canonical missing labels", func(t *testing.T) {
		plan := `# Goal
- [ ] incomplete outcome
  - Files: internal/example.go
Final Verify:
`
		got := strings.Join(planContractFindings(plan), "\n")
		for _, want := range []string{"`Change:`", "`Verify:`", "non-empty `Final Verify:`"} {
			if !strings.Contains(got, want) {
				t.Errorf("findings %q missing %q", got, want)
			}
		}
	})
}
