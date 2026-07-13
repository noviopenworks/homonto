# Tasks — doctor-reports-catalog-upgrade
## 1. Doctor catalog-version finding
- [x] Add catalogUpgradeFinding(recorded, embedded) helper; Doctor reports a
      finding when they differ. Test the helper (differ -> finding; equal -> none).
## 2. Verify
- [x] go test ./internal/engine/... ./internal/cli/... -race, vet, build, validate --all green.
