# Tasks — close-archive-rollback

## 1. Roll back archived flag on move failure
- [x] runClose rolls st.Archived back to false (re-save) if MkdirAll/Rename
      fails. TDD: an injected rename failure leaves archived=false and the
      change unmoved; success path unchanged.

## 2. Verify
- [x] `go test ./internal/ontocli/... -race`, vet, build, `openspec validate
      --all` green.
