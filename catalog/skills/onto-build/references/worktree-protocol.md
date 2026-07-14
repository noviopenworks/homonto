# Worktree isolation protocol (`isolation: worktree`)

`onto set isolation <name> worktree` records the *choice*; this is the *how*.
Worktree isolation gives a change its own working directory + branch, so parallel
work (or a dirty current branch) never contaminates it. Prefer it over a plain
branch when the current tree is dirty, when several changes are active at once, or
when build dispatches parallel implementers.

## 0. Detect existing isolation first

If a native worktree/sandbox tool is available, use it — it places the directory,
creates the branch, and cleans up, and its state is visible to the harness. Using
raw `git worktree add` when a native tool exists creates phantom state the harness
can't manage. Check whether you are already in a worktree (`git rev-parse
--git-common-dir` differs from `--git-dir`) before creating another.

## 1. Create the workspace (git fallback)

```sh
git worktree add "<path>" -b "<type>/YYYYMMDD/<change-name>"
cd "<path>"
```

Path: a sibling dir outside the repo (e.g. `../<repo>-worktrees/<name>`) or a
project-local ignored dir. If `git worktree add` fails on a sandbox permission
error, tell the user the sandbox blocked it and fall back to working in place on a
branch (record `isolation: branch`).

## 2. Set up and baseline

Reproduce the project's environment in the new tree (install deps, copy any
untracked but required local config/`.env` the build needs — a worktree does NOT
inherit untracked files), then run the build + test suite once to confirm a
**clean baseline** before the first task. Building on an already-red tree hides
which failure you introduced.

## 3. Work, then integrate

Do the change's build in the worktree, one commit per task on its branch. At
close, the `integration` choice (`merge`/`pr`) integrates the branch (see
onto-close).

## 4. Clean up

After the change is closed and integrated, remove the worktree so it doesn't
linger as phantom state:

```sh
git worktree remove "<path>"      # or the native tool's teardown
git worktree prune
```

Never leave an orphaned worktree pointing at a merged/deleted branch.
