package tocli

import (
	"fmt"
	"regexp"
	"strings"
)

const finalVerifyLabel = "Final Verify:"

// checkboxLine and uncheckedTask recognize the task markers shared by doctor,
// handoff, and the to-do resume loop.
var (
	requiredTaskFields = []string{"Files", "Change", "Verify"}
	checkboxLine       = regexp.MustCompile(`^\s*[-*] \[[ x]\] `)
	uncheckedTask      = regexp.MustCompile(`^\s*[-*] \[ \] `)
	taskContractField  = regexp.MustCompile(`^\s+[-*]\s+(` + strings.Join(requiredTaskFields, "|") + `):\s*\S`)
	finalVerifyLine    = regexp.MustCompile(`^\s*` + regexp.QuoteMeta(finalVerifyLabel) + `\s*\S`)
)

type planTaskBlock struct {
	start int
	end   int
}

func planTaskBlocks(lines []string) []planTaskBlock {
	blocks := []planTaskBlock{}
	for i := 0; i < len(lines); i++ {
		if !checkboxLine.MatchString(lines[i]) {
			continue
		}
		indent := leadingIndent(lines[i])
		end := i + 1
		for end < len(lines) {
			if strings.TrimSpace(lines[end]) != "" && leadingIndent(lines[end]) <= indent {
				break
			}
			end++
		}
		blocks = append(blocks, planTaskBlock{start: i, end: end})
		i = end - 1
	}
	return blocks
}

func leadingIndent(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

func hasUncheckedTask(plan string) bool {
	for _, line := range strings.Split(plan, "\n") {
		if uncheckedTask.MatchString(line) {
			return true
		}
	}
	return false
}

// planContractFindings checks only the lightweight Markdown contract that the
// skills consume. It returns diagnostics; callers never use it as a phase gate.
func planContractFindings(plan string) []string {
	lines := strings.Split(plan, "\n")
	blocks := planTaskBlocks(lines)
	if len(blocks) == 0 {
		return []string{"at do, but plan.md has no `- [ ]` task checkboxes — the to-do resume logic cannot track completion"}
	}

	findings := []string{}
	for _, block := range blocks {
		present := map[string]bool{}
		for _, line := range lines[block.start+1 : block.end] {
			match := taskContractField.FindStringSubmatch(line)
			if len(match) > 1 {
				present[match[1]] = true
			}
		}
		missing := []string{}
		for _, field := range requiredTaskFields {
			if !present[field] {
				missing = append(missing, fmt.Sprintf("`%s:`", field))
			}
		}
		if len(missing) > 0 {
			findings = append(findings, fmt.Sprintf("plan task on line %d is missing %s", block.start+1, strings.Join(missing, ", ")))
		}
	}

	hasFinalVerify := false
	for _, line := range lines {
		if finalVerifyLine.MatchString(line) {
			hasFinalVerify = true
			break
		}
	}
	if !hasFinalVerify {
		findings = append(findings, "plan.md has no non-empty `Final Verify:` line")
	}
	return findings
}
