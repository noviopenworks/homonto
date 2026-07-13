# Tasks — import-backup-before-overwrite
## 1. Backup + atomic overwrite on --force
- [ ] Before overwriting an existing config with --force, copy it to <config>.bak;
      write the new config atomically. Test: --force over an existing config leaves
      a .bak with the old contents and writes the new config.
## 2. Verify
- [ ] `go test ./internal/cli/... -race`, vet, build, `openspec validate --all` green.
