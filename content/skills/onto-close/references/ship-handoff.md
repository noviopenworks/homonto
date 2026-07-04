# Ship handoff contract

onto ends at a closed change — PR creation and review stay outside the
workflow. Close *prepares* the handoff so the PR skills start with
everything they need.

## After archive, offer

Present a ready PR body assembled from the archived change:

```markdown
## <change title, from the proposal>

<proposal Why, condensed to 1–2 paragraphs>

### What changed

<proposal What Changes bullets, updated to what actually shipped>

### Verification

<verification.md summary: mode, Result, scenario count, adversarial pass
outcome, regression result>

Full records: `docs/changes/archive/YYYY-MM-DD-<name>/`
(proposal · design · verification · notes)
```

## If the user accepts

1. Write the body to `docs/changes/archive/YYYY-MM-DD-<name>/ship.md`
   and commit — this is the archive contract's single sanctioned
   post-archive addition (`docs/changes/README.md`).
2. Name the next step explicitly: the dedicated PR/commit-push skills take
   it from here — onto does not push, does not open PRs.

## If declined

Nothing is written; the offer is not repeated **unprompted** — a later
user request may still produce `ship.md` (the sanctioned archive exception
is the file itself, not its timing).

## If no PR skills exist

Hand the assembled body directly to the user as the deliverable; the
handoff contract is the body, not any particular skill.
