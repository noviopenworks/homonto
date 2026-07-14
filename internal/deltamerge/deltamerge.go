// Package deltamerge applies an onto delta spec onto a living capability spec,
// deterministically — the RENAMED → MODIFIED → REMOVED → ADDED merge that
// onto-close otherwise performs by hand (the workflow's most destructive step).
//
// A living spec is `## Requirements` followed by `### Requirement: <name>`
// blocks. A delta groups those blocks under `## ADDED|MODIFIED|REMOVED|RENAMED
// Requirements` sections. Merge applies the four in the fixed order so a
// MODIFIED targeting a just-RENAMED name resolves, then returns the merged
// living spec — carrying no delta-only section headings and no duplicate
// requirement name (Lint verifies both).
package deltamerge

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reqHeading     = regexp.MustCompile(`^### Requirement:\s*(.+?)\s*$`)
	sectionHeading = regexp.MustCompile(`^##\s+(ADDED|MODIFIED|REMOVED|RENAMED)\s+Requirements\s*$`)
	topHeading     = regexp.MustCompile(`^##\s+`)
	fromLine       = regexp.MustCompile(`^\s*-?\s*FROM:\s*(.+?)\s*$`)
	toLine         = regexp.MustCompile(`^\s*TO:\s*(.+?)\s*$`)
)

type requirement struct {
	name  string
	block []string // lines, starting with the "### Requirement: <name>" heading
}

// Merge applies delta onto living (the current living spec for capability, or ""
// when none exists) and returns the merged living spec. It errors when a
// MODIFIED/REMOVED name or a RENAMED FROM is absent, or an ADDED name already
// exists — the delta references reality that isn't there.
func Merge(capability, living, delta string) (string, error) {
	renamed, modified, removed, added, err := parseDelta(delta)
	if err != nil {
		return "", err
	}
	preamble, reqs := splitLiving(capability, living)

	find := func(name string) int {
		for i, r := range reqs {
			if r.name == name {
				return i
			}
		}
		return -1
	}

	// 1. RENAMED — rename the heading, keep the body.
	for _, ft := range renamed {
		i := find(ft[0])
		if i < 0 {
			return "", fmt.Errorf("deltamerge: RENAMED FROM %q not found in living spec %q", ft[0], capability)
		}
		reqs[i].name = ft[1]
		reqs[i].block[0] = "### Requirement: " + ft[1]
	}
	// 2. MODIFIED — replace the whole requirement.
	for _, m := range modified {
		i := find(m.name)
		if i < 0 {
			return "", fmt.Errorf("deltamerge: MODIFIED %q not found in living spec %q", m.name, capability)
		}
		reqs[i] = m
	}
	// 3. REMOVED — delete the named requirement.
	for _, rm := range removed {
		i := find(rm.name)
		if i < 0 {
			return "", fmt.Errorf("deltamerge: REMOVED %q not found in living spec %q", rm.name, capability)
		}
		reqs = append(reqs[:i], reqs[i+1:]...)
	}
	// 4. ADDED — append; a name that already exists is a conflict.
	for _, a := range added {
		if find(a.name) >= 0 {
			return "", fmt.Errorf("deltamerge: ADDED %q already exists in living spec %q", a.name, capability)
		}
		reqs = append(reqs, a)
	}

	return assemble(preamble, reqs), nil
}

// Lint checks a merged living spec for the two failure modes the merge must
// never produce: a leaked delta-only section heading, and a duplicated
// requirement name.
func Lint(merged string) []string {
	var findings []string
	seen := map[string]bool{}
	for _, ln := range strings.Split(merged, "\n") {
		if sectionHeading.MatchString(ln) {
			findings = append(findings, "leaked delta section heading: "+strings.TrimSpace(ln))
		}
		if m := reqHeading.FindStringSubmatch(ln); m != nil {
			if seen[m[1]] {
				findings = append(findings, "duplicated requirement: "+m[1])
			}
			seen[m[1]] = true
		}
	}
	return findings
}

// splitLiving separates the living spec into its preamble (title/prose through
// the `## Requirements` heading) and its requirement blocks. For an empty living
// spec it synthesizes a minimal preamble titled from the capability.
func splitLiving(capability, living string) (preamble []string, reqs []requirement) {
	if strings.TrimSpace(living) == "" {
		return []string{"# " + capability, "", "## Requirements"}, nil
	}
	lines := strings.Split(strings.ReplaceAll(living, "\r\n", "\n"), "\n")
	i := 0
	for i < len(lines) && !reqHeading.MatchString(lines[i]) {
		preamble = append(preamble, lines[i])
		i++
	}
	for i < len(lines) {
		if m := reqHeading.FindStringSubmatch(lines[i]); m != nil {
			block := []string{lines[i]}
			i++
			for i < len(lines) && !reqHeading.MatchString(lines[i]) && !topHeading.MatchString(lines[i]) {
				block = append(block, lines[i])
				i++
			}
			reqs = append(reqs, requirement{name: m[1], block: trimBlock(block)})
		} else {
			i++ // a trailing top-level section after requirements — dropped (rare)
		}
	}
	return preamble, reqs
}

// parseDelta reads the four delta sections. ADDED/MODIFIED yield requirement
// blocks; REMOVED yields names; RENAMED yields FROM/TO pairs.
func parseDelta(delta string) (renamed [][2]string, modified, removed []requirement, added []requirement, err error) {
	// note: removed carries names in requirement.name (block unused).
	lines := strings.Split(strings.ReplaceAll(delta, "\r\n", "\n"), "\n")
	section := ""
	var pendingFrom string
	i := 0
	for i < len(lines) {
		ln := lines[i]
		if m := sectionHeading.FindStringSubmatch(ln); m != nil {
			section = m[1]
			pendingFrom = ""
			i++
			continue
		}
		switch section {
		case "ADDED", "MODIFIED":
			if m := reqHeading.FindStringSubmatch(ln); m != nil {
				block := []string{ln}
				i++
				for i < len(lines) && !reqHeading.MatchString(lines[i]) && !topHeading.MatchString(lines[i]) {
					block = append(block, lines[i])
					i++
				}
				r := requirement{name: m[1], block: trimBlock(block)}
				if section == "ADDED" {
					added = append(added, r)
				} else {
					modified = append(modified, r)
				}
				continue
			}
		case "REMOVED":
			if m := reqHeading.FindStringSubmatch(ln); m != nil {
				removed = append(removed, requirement{name: m[1]})
			}
		case "RENAMED":
			if m := fromLine.FindStringSubmatch(ln); m != nil {
				pendingFrom = m[1]
			} else if m := toLine.FindStringSubmatch(ln); m != nil {
				if pendingFrom == "" {
					return nil, nil, nil, nil, fmt.Errorf("deltamerge: RENAMED TO %q has no preceding FROM", m[1])
				}
				renamed = append(renamed, [2]string{pendingFrom, m[1]})
				pendingFrom = ""
			}
		}
		i++
	}
	return renamed, modified, removed, added, nil
}

// trimBlock removes trailing blank lines from a requirement block.
func trimBlock(block []string) []string {
	for len(block) > 0 && strings.TrimSpace(block[len(block)-1]) == "" {
		block = block[:len(block)-1]
	}
	return block
}

// assemble reconstitutes the living spec: the preamble, then each requirement
// block, separated by a single blank line, ending with one newline.
func assemble(preamble []string, reqs []requirement) string {
	var b strings.Builder
	pre := strings.TrimRight(strings.Join(preamble, "\n"), "\n")
	b.WriteString(pre)
	b.WriteString("\n")
	for _, r := range reqs {
		b.WriteString("\n")
		b.WriteString(strings.Join(r.block, "\n"))
		b.WriteString("\n")
	}
	return b.String()
}
