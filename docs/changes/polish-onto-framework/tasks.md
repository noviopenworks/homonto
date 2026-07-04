# Tasks: polish-onto-framework

## 1. Foundation

- [ ] 1.1 Artifact templates authored (state.yaml, proposal, design, tasks,
      plan, verification, ADR draft) in the location the design decides
- [x] 1.2 Layout contracts updated to reference the templates and the new
      state.yaml fields (deps, metrics)

## 2. Skill upgrades

- [x] 2.1 onto (dispatcher): deps/blocked awareness, parallel-change
      guidance, template pointers
- [x] 2.2 onto-open: notes.md checkpoint protocol, template usage
- [ ] 2.3 onto-design: notes.md checkpoint, optional parallel approach
      exploration (subagents), template usage
- [ ] 2.4 onto-build: subagent execution protocol (implementer + reviewer),
      template usage
- [ ] 2.5 onto-verify: adversarial multi-agent verification protocol
- [ ] 2.6 onto-close: format lint, RENAMED merge semantics, metrics stamp,
      ship handoff contract
- [ ] 2.7 onto-fix / onto-tweak: preset-appropriate versions of the above

## 3. Docs

- [ ] 3.1 Guide updated (docs/guides/onto-workflow.md) for all seven axes
- [ ] 3.2 ADR draft(s) for the significant decisions

## 4. Validation

- [ ] 4.1 Dry-run: full lifecycle with new templates + checkpoints +
      subagent build protocol (fresh-context agent)
- [ ] 4.2 Dry-run: adversarial verify + close lint + metrics + deps
      scenarios (fresh-context agent)
- [ ] 4.3 Self-containment + table-sync + template-conformance checks;
      regression (go test) stays green
