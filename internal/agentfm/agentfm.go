// Package agentfm renders per-tool subagent frontmatter from one neutral source.
//
// Claude Code and OpenCode express a subagent's tool access differently: Claude
// uses a `tools:` allowlist string, OpenCode uses a `permission:` map — and the
// two cannot coexist in one file (OpenCode rejects a string `tools:`). So a
// subagent whose access must be enforced in BOTH tools declares its intent once,
// tool-neutrally, in a `homonto:` frontmatter block:
//
//	---
//	name: code-reviewer
//	description: ...
//	mode: subagent
//	homonto:
//	  read_only: true   # deny edits/writes
//	  bash: false       # optional; false denies bash too (default: allowed)
//	  dialogs: true     # allow the interactive question/dialog tool
//	---
//	<prompt body>
//
// Render() rewrites that block into each tool's native fields, preserving every
// other frontmatter line verbatim (so name/description are never re-quoted) and
// the body byte-for-byte. Subagents without a `homonto:` block are returned
// unchanged.
package agentfm

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Homonto is the neutral access intent declared under the `homonto:` key.
type Homonto struct {
	ReadOnly bool  `yaml:"read_only"`
	Bash     *bool `yaml:"bash"` // nil = default (allowed); false = deny
	Dialogs  bool  `yaml:"dialogs"`
}

// NeedsTransform reports whether content carries a `homonto:` frontmatter block
// (and therefore must be rendered per tool rather than projected verbatim).
func NeedsTransform(content []byte) bool {
	fm, _, ok := split(content)
	if !ok {
		return false
	}
	_, has := parseHomonto(fm)
	return has
}

// Render returns content rewritten for tool ("claude" or "opencode"). If content
// has no frontmatter or no `homonto:` block, it is returned unchanged.
func Render(content []byte, tool string) ([]byte, error) {
	fm, body, ok := split(content)
	if !ok {
		return content, nil
	}
	h, has := parseHomonto(fm)
	if !has {
		return content, nil
	}

	kept := stripHomontoBlock(fm)
	var extra string
	switch tool {
	case "claude":
		extra = "tools: " + claudeTools(h)
	case "opencode":
		extra = "permission:\n" + opencodePermission(h)
	default:
		return nil, fmt.Errorf("agentfm: unknown tool %q", tool)
	}

	var b bytes.Buffer
	b.WriteString("---\n")
	for _, ln := range kept {
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	b.WriteString(extra)
	b.WriteString("\n---\n")
	b.Write(body)
	return b.Bytes(), nil
}

// claudeTools renders the Claude `tools:` allowlist for a read-only agent: the
// read tools always, plus Bash unless it is explicitly denied. A non-read_only
// agent (unusual for a homonto block) still gets edit/write tools.
func claudeTools(h Homonto) string {
	tools := []string{"Read", "Grep", "Glob"}
	if h.Bash == nil || *h.Bash {
		tools = append(tools, "Bash")
	}
	if !h.ReadOnly {
		tools = append(tools, "Edit", "Write")
	}
	return strings.Join(tools, ", ")
}

// opencodePermission renders the OpenCode `permission:` block body (indented
// lines) for the neutral intent.
func opencodePermission(h Homonto) string {
	var lines []string
	if h.ReadOnly {
		lines = append(lines, "  edit: deny")
	}
	if h.Bash != nil && !*h.Bash {
		lines = append(lines, "  bash: deny")
	}
	if h.Dialogs {
		lines = append(lines, "  question: allow")
	}
	return strings.Join(lines, "\n")
}

// split separates content into its frontmatter lines and the remaining body.
// ok is false when content does not open with a `---` frontmatter fence.
func split(content []byte) (fm []byte, body []byte, ok bool) {
	if !bytes.HasPrefix(content, []byte("---\n")) {
		return nil, nil, false
	}
	rest := content[len("---\n"):]
	fm, body, found := bytes.Cut(rest, []byte("\n---\n"))
	if !found {
		return nil, nil, false
	}
	return fm, body, true
}

// parseHomonto reads the `homonto:` block from frontmatter YAML.
func parseHomonto(fm []byte) (Homonto, bool) {
	var doc struct {
		Homonto *Homonto `yaml:"homonto"`
	}
	if err := yaml.Unmarshal(fm, &doc); err != nil || doc.Homonto == nil {
		return Homonto{}, false
	}
	return *doc.Homonto, true
}

// stripHomontoBlock returns the frontmatter lines with the `homonto:` key and its
// indented child lines removed, and comment-only lines dropped (the catalog's
// homonto comments are maintainer notes that must not leak into the already-
// rendered projected file). Every other line is preserved verbatim.
func stripHomontoBlock(fm []byte) []string {
	var out []string
	lines := strings.Split(string(fm), "\n")
	skipping := false
	for _, ln := range lines {
		if skipping {
			// Child lines of the block are indented; the first non-indented,
			// non-blank line ends the block.
			if strings.TrimSpace(ln) == "" || ln[0] == ' ' || ln[0] == '\t' {
				continue
			}
			skipping = false
		}
		if ln == "homonto:" || strings.HasPrefix(ln, "homonto:") {
			skipping = true
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(ln), "#") {
			continue
		}
		out = append(out, ln)
	}
	return out
}
