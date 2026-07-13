# Tasks — control-plane-fs-safety-locking
## 1. F25 no-follow control-plane writer
- [ ] Add a no-follow, root-confined write function for `.homonto` control-plane
      paths; refuse writing through a symlink and outside the `.homonto` root.
      Route state/cache/lock/catalog writes through it. Test: a symlinked target
      is refused, a normal write succeeds.
## 2. F29 project apply lock
- [ ] Acquire a project-scoped exclusive lock (`.homonto/apply.lock`) at apply
      start; a second concurrent apply fails fast; released on exit. Test.
## 3. F31 locator credential redaction
- [ ] Reject or redact credentials in remote locators before they reach errors or
      `remote.lock.json`. Test: a `https://user:pass@host/…` locator does not leak
      the credential into the lockfile or an error string.
## 4. Verification
- [ ] `go test ./internal/... -race`, vet, build, `openspec validate --all` green.
