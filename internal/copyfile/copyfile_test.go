package copyfile

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, p string, b []byte) {
	t.Helper()
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// find returns the single Op for dst, failing if absent.
func find(t *testing.T, ops []Op, dst string) Op {
	t.Helper()
	for _, o := range ops {
		if o.Dst == dst {
			return o
		}
	}
	t.Fatalf("no op for %s in %+v", dst, ops)
	return Op{}
}

func TestPlanCreateNoopUpdate(t *testing.T) {
	dir := t.TempDir()
	create := filepath.Join(dir, "create.md")
	noop := filepath.Join(dir, "noop.md")
	update := filepath.Join(dir, "update.md")

	write(t, noop, []byte("same"))
	write(t, update, []byte("old")) // ours, unchanged since we wrote "old"

	desired := map[string][]byte{
		create: []byte("new"),
		noop:   []byte("same"),
		update: []byte("newer"),
	}
	recorded := map[string]string{
		noop:   Hash([]byte("same")),
		update: Hash([]byte("old")),
	}
	ops, err := Plan(desired, recorded)
	if err != nil {
		t.Fatal(err)
	}
	if got := find(t, ops, create).Action; got != Create {
		t.Fatalf("create: %s", got)
	}
	if got := find(t, ops, noop).Action; got != Noop {
		t.Fatalf("noop: %s", got)
	}
	if got := find(t, ops, update).Action; got != Update {
		t.Fatalf("update: %s", got)
	}
}

func TestPlanLocalEdit(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "edited.md")
	write(t, p, []byte("user edited")) // on disk differs from recorded base
	ops, err := Plan(
		map[string][]byte{p: []byte("upstream")},
		map[string]string{p: Hash([]byte("base"))},
	)
	if err != nil {
		t.Fatal(err)
	}
	op := find(t, ops, p)
	if op.Action != LocalEdit {
		t.Fatalf("want local-edit, got %s", op.Action)
	}
	if string(op.OnDisk) != "user edited" || op.Recorded != Hash([]byte("base")) {
		t.Fatalf("local-edit must carry on-disk bytes + recorded base: %+v", op)
	}
}

func TestPlanForeignFileConflict(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "foreign.md")
	write(t, p, []byte("hand written")) // exists, NO record of ownership
	ops, err := Plan(map[string][]byte{p: []byte("ours")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := find(t, ops, p).Action; got != Conflict {
		t.Fatalf("a foreign file must be a conflict, got %s", got)
	}
}

func TestPlanSymlinkConflict(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	write(t, target, []byte("x"))
	p := filepath.Join(dir, "link.md")
	if err := os.Symlink(target, p); err != nil {
		t.Fatal(err)
	}
	// Even if recorded, a symlink at a copy destination is not ours to overwrite.
	ops, err := Plan(map[string][]byte{p: []byte("ours")}, map[string]string{p: Hash([]byte("x"))})
	if err != nil {
		t.Fatal(err)
	}
	if got := find(t, ops, p).Action; got != Conflict {
		t.Fatalf("a symlink at a copy dst must be a conflict, got %s", got)
	}
}

func TestPlanPruneAndPruneLocalEdit(t *testing.T) {
	dir := t.TempDir()
	prune := filepath.Join(dir, "prune.md")
	kept := filepath.Join(dir, "kept.md")
	write(t, prune, []byte("ours unchanged"))
	write(t, kept, []byte("user changed it"))

	recorded := map[string]string{
		prune: Hash([]byte("ours unchanged")),
		kept:  Hash([]byte("ours original")),
	}
	ops, err := Plan(map[string][]byte{}, recorded) // nothing desired → prune both records
	if err != nil {
		t.Fatal(err)
	}
	if got := find(t, ops, prune).Action; got != Prune {
		t.Fatalf("an unchanged de-declared managed file must prune, got %s", got)
	}
	if got := find(t, ops, kept).Action; got != LocalEdit {
		t.Fatalf("a de-declared but user-edited file must NOT prune (local-edit), got %s", got)
	}
}

func TestPlanAbsentPrunedRecordIsSkipped(t *testing.T) {
	dir := t.TempDir()
	gone := filepath.Join(dir, "gone.md")
	ops, err := Plan(map[string][]byte{}, map[string]string{gone: Hash([]byte("whatever"))})
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 0 {
		t.Fatalf("an already-gone recorded file needs no op, got %+v", ops)
	}
}
