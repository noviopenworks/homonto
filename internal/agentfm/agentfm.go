// Package agentfm renders per-tool subagent frontmatter from one neutral source.
//
// Claude Code and OpenCode express an agent's capabilities differently (Claude:
// a `tools:` allowlist string; OpenCode: a `permission:` map and `mode`), and the
// two cannot coexist in one file (OpenCode rejects a string `tools:`). So an
// agent declares its intent once, tool-neutrally, in a `homonto:` frontmatter
// block, and Render() emits each tool's native dialect:
//
//	---
//	name: onto-reviewer
//	description: ...
//	mode: subagent
//	homonto:
//	  role: architectural   # model tier → stamped from [models.<tool>.<role>]
//	  read_only: true       # deny edits/writes
//	  bash: false           # optional; false denies bash (default: allowed)
//	  dialogs: true          # allow the interactive question/dialog tool
//	  spawn: []             # delegation topology: agents this one may dispatch
//	  primary: true         # OpenCode primary agent; SKIPPED for Claude
//	  steps: 60             # OpenCode iteration budget
//	---
//	<prompt body>
//
// Parity is by explicit tiers: read_only/bash/role/spawn:[] render fully in both
// tools; a named spawn list is enforced in OpenCode (task globs) and advisory in
// Claude (Task present); primary/steps are OpenCode-only (Render returns nil for
// the Claude variant of a primary agent — its entry point is the /onto command).
// Every non-homonto frontmatter line except `mode:` is preserved verbatim.
package agentfm

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// TierNames are the model tiers an agent's `role:` may declare — the same four
// levels a [models.<tool>.<route>] block must define, in the order validation
// reports them: architectural (orchestrate/design), coding (implement), review
// (judge others' work — the reviewer and the skeptic), trivial (cheap
// lookups). Single source of truth, like ClaudeAliases: config validation and
// rendering both reference it, so an unknown tier fails loudly in both places
// instead of silently rendering an agent with no model.
var TierNames = []string{"architectural", "coding", "review", "trivial"}

// Tiers is TierNames as a membership set.
var Tiers = func() map[string]bool {
	m := make(map[string]bool, len(TierNames))
	for _, t := range TierNames {
		m[t] = true
	}
	return m
}()

// Homonto is the neutral capability intent declared under the `homonto:` key.
type Homonto struct {
	Role     string    `yaml:"role"`      // "" or a Tiers key → model
	ReadOnly bool      `yaml:"read_only"` // deny edits/writes
	Bash     *bool     `yaml:"bash"`      // nil = default (allowed); false = deny
	Dialogs  bool      `yaml:"dialogs"`   // allow the question/dialog tool
	Spawn    *[]string `yaml:"spawn"`     // nil = unrestricted; [] = none; [a,b] = only these
	Primary  bool      `yaml:"primary"`   // OpenCode primary agent (Claude: skip)
	Steps    int       `yaml:"steps"`     // OpenCode iteration budget
}

// ModelSpec is a fully-resolved model choice for one tool: which model, which
// variant of it, and how hard to think. Each tool spells these differently —
// see Render — so they are carried neutrally and rendered per tool.
type ModelSpec struct {
	Model   string
	Variant string
	Effort  string
}

// merge returns s with every non-empty field of ov overriding it, so a
// per-subagent block can override just `effort` and inherit the tier's model.
func (s ModelSpec) merge(ov ModelSpec) ModelSpec {
	if ov.Model != "" {
		s.Model = ov.Model
	}
	if ov.Variant != "" {
		s.Variant = ov.Variant
	}
	if ov.Effort != "" {
		s.Effort = ov.Effort
	}
	return s
}

// RenderContext carries the config-derived values the render needs for the tool
// being rendered (the caller passes the Claude values for the claude render, the
// OpenCode values for the opencode render).
//
// Roles is the role→spec map from [models.<tool>.<role>] — the default for any
// agent declaring that role. Overrides is keyed by subagent name and wins field
// by field, so [subagents.<name>.<tool>] can retune one agent without restating
// its tier.
type RenderContext struct {
	Roles     map[string]ModelSpec
	Overrides map[string]ModelSpec
}

// specFor resolves the model spec for an agent: its role's tier default, with
// any per-subagent override applied field by field.
func (c RenderContext) specFor(name, role string) ModelSpec {
	return c.Roles[role].merge(c.Overrides[name])
}

// ClaudeAliases are the model aliases Claude Code accepts. The bracketed
// variant syntax (`opus[1m]`) is documented for aliases ONLY — a full model id
// such as claude-opus-4-8 takes no variant. This is the single source of truth;
// config validation references it rather than keeping a copy that could drift.
var ClaudeAliases = map[string]bool{
	"opus": true, "sonnet": true, "haiku": true, "fable": true, "opusplan": true,
}

// ClaudeEffortLevels are the values Claude Code's agent `effort:` field
// accepts. Single source of truth, same as ClaudeAliases.
var ClaudeEffortLevels = map[string]bool{
	"low": true, "medium": true, "high": true, "xhigh": true, "max": true,
}

// claudeModel renders the Claude `model:` value. Claude has no separate variant
// field: a variant is expressed by bracketing the alias. A variant on a
// non-alias model has no Claude spelling at all — that is an ERROR here, never
// a silent drop: the merged (tier + override) model isn't known until render,
// so load-time validation cannot always catch the combination, and silently
// dropping the variant would ship an agent quietly weaker than declared.
func claudeModel(s ModelSpec) (string, error) {
	if s.Model == "" || s.Variant == "" {
		return s.Model, nil
	}
	if !ClaudeAliases[s.Model] {
		return "", fmt.Errorf("variant %q needs a model alias (opus, sonnet, haiku, fable, opusplan) — Claude takes no variant on the full model id %q", s.Variant, s.Model)
	}
	return s.Model + "[" + s.Variant + "]", nil
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

// ProjectsFor reports whether content is projected for tool at all. It is false
// only where Render deliberately emits nothing — the Claude variant of an
// OpenCode-primary agent. Callers use it to tell "deliberately not projected
// here" apart from "should be here and is missing", so a by-design absence is
// never reported as a fixable finding.
func ProjectsFor(content []byte, tool string) (bool, error) {
	// Projection is decided by the neutral block alone (primary vs not), never by
	// the model spec, so an empty context is the right question to ask here.
	rendered, err := Render("", content, tool, RenderContext{})
	if err != nil {
		return false, err
	}
	return rendered != nil, nil
}

// Render returns content rewritten for tool ("claude" or "opencode"), or nil
// bytes when the agent must NOT be projected for that tool (a primary agent has
// no Claude variant). Content with no frontmatter or no `homonto:` block is
// returned unchanged.
func Render(name string, content []byte, tool string, ctx RenderContext) ([]byte, error) {
	fm, body, ok := split(content)
	if !ok {
		return content, nil
	}
	h, has := parseHomonto(fm)
	if !has {
		return content, nil
	}
	// An unknown role would look up no tier and render the agent with no model
	// line at all — a silently weaker agent. Fail loudly instead, naming the
	// agent and the valid tiers.
	if h.Role != "" && !Tiers[h.Role] {
		return nil, fmt.Errorf("agentfm: agent %q: unknown role %q; valid roles are %s", name, h.Role, strings.Join(TierNames, ", "))
	}
	spec := ctx.specFor(name, h.Role)

	// Preserve every frontmatter line except the homonto block and the mode line
	// (re-emitted per tool below).
	var kept []string
	for _, ln := range stripHomontoBlock(fm) {
		if strings.HasPrefix(strings.TrimSpace(ln), "mode:") {
			continue
		}
		kept = append(kept, ln)
	}

	var extra []string
	switch tool {
	case "claude":
		if h.Primary {
			return nil, nil // Claude has no primary-agent concept; entry is /onto
		}
		extra = append(extra, "mode: subagent", "tools: "+claudeTools(h))
		// Claude carries the variant inside the model string (`opus[1m]`) and
		// effort as its own frontmatter field.
		m, merr := claudeModel(spec)
		if merr != nil {
			return nil, fmt.Errorf("agentfm: agent %q: %w", name, merr)
		}
		if m != "" {
			extra = append(extra, "model: "+m)
		}
		if spec.Effort != "" {
			extra = append(extra, "effort: "+spec.Effort)
		}
	case "opencode":
		mode := "subagent"
		if h.Primary {
			mode = "primary"
		}
		extra = append(extra, "mode: "+mode)
		// OpenCode is the mirror image: `variant` is its own field, and there is
		// no effort concept at all — dropping it here is why the config layer
		// reports the drop once at plan time rather than failing.
		if spec.Model != "" {
			extra = append(extra, "model: "+spec.Model)
		}
		if spec.Variant != "" {
			extra = append(extra, "variant: "+spec.Variant)
		}
		if h.Steps > 0 {
			extra = append(extra, fmt.Sprintf("steps: %d", h.Steps))
		}
		if perm := opencodePermission(h); perm != "" {
			extra = append(extra, "permission:", perm)
		}
	default:
		return nil, fmt.Errorf("agentfm: unknown tool %q", tool)
	}

	var b bytes.Buffer
	b.WriteString("---\n")
	for _, ln := range kept {
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	for _, ln := range extra {
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	b.WriteString("---\n")
	b.Write(body)
	return b.Bytes(), nil
}

// claudeTools renders the Claude `tools:` allowlist. Claude models capability as
// an allowlist (not a deny map), so this enumerates what the agent MAY use:
// read tools always; Bash unless denied; Edit/Write unless read_only; Task
// unless the agent declares it may spawn nothing (spawn: []).
func claudeTools(h Homonto) string {
	tools := []string{"Read", "Grep", "Glob"}
	if h.Bash == nil || *h.Bash {
		tools = append(tools, "Bash")
	}
	if !h.ReadOnly {
		tools = append(tools, "Edit", "Write")
	}
	// spawn nil → unrestricted (Task allowed); spawn [] → no spawning (omit Task);
	// spawn [named] → Task allowed (Claude cannot scope to specific agents).
	if h.Spawn == nil || len(*h.Spawn) > 0 {
		tools = append(tools, "Task")
	}
	return strings.Join(tools, ", ")
}

// opencodePermission renders the OpenCode `permission:` block body (indented
// lines) for the neutral intent, including the delegation topology as task globs.
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
	if h.Spawn != nil {
		if len(*h.Spawn) == 0 {
			lines = append(lines, "  task: deny")
		} else {
			lines = append(lines, "  task:", `    "*": deny`)
			for _, a := range *h.Spawn {
				lines = append(lines, fmt.Sprintf("    %q: allow", a))
			}
		}
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
