# Tasks — init-augment-existing-gitignore
## 1. Augment existing .gitignore
- [x] Init appends missing /.homonto/ and .env to an existing .gitignore, preserving
      content; reports created vs updated. Test covers augmentation + preservation.
## 2. Verify
- [x] go test scaffold+cli -race, vet, build, openspec validate --all green.
