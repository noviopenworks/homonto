# Tasks: address-deep-review

## 1. Product code (TDD: failing test first per bug)

- [x] 1.1 Claude MCP schema fix + conformance fixtures from real tool
      output; import reads string+args (legacy array tolerated) with
      expanded redaction
- [x] 1.2 Secret safety: unknown-provenance redaction in plan; shared
      mode-preserving fsync'd writeAtomic (0600 default, unique temps);
      memoized resolver
- [x] 1.3 Pruning: delete action end-to-end (plan/apply/state/render/
      HasChanges), skill links recorded in state and pruned safely
- [ ] 1.4 Robustness: sjson path escaping everywhere, skill-name
      validation at config load, sorted deterministic plans, non-object
      root error

## 2. Hygiene

- [ ] 2.1 MIT LICENSE, GitHub Actions CI (vet+test), var Version,
      README honesty pass

## 3. onto v2.1

- [ ] 3.1 tweak covers small features; preflight warns not halts;
      close rewrites ADR links pre-archive; ADR 0007 errata; guide +
      dispatcher table sync; ADR draft for preflight decision

## 4. Specs

- [ ] 4.1 Delta specs: tool-adapters, apply-pipeline, cli-commands,
      secret-references, onto-workflow

## 5. Validation

- [ ] 5.1 Full suite green + every review reproduction re-run and
      shown fixed; fresh-context skeptics at verify (full mode)
