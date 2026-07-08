package link

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// managed reports whether target points inside contentRoot — the content homonto
// owns. A symlink pointing there is one of ours and may be relinked or pruned; a
// symlink pointing anywhere else is user-owned and must never be touched.
func managed(target, contentRoot string) bool {
	return strings.HasPrefix(target, contentRoot+string(os.PathSeparator))
}

// Link ensures dst is a symlink to src, returning whether it changed. A regular
// file (or dir) at dst is never clobbered — that is a "conflict" error. A
// symlink already pointing at src is a no-op. A symlink pointing elsewhere is
// relinked in place only when it is ours (its target sits inside contentRoot);
// a symlink pointing outside contentRoot is a foreign, user-owned link and is a
// conflict — homonto must never remove or repoint what it does not own.
func Link(src, dst, contentRoot string) (bool, error) {
	if fi, err := os.Lstat(dst); err == nil {
		if fi.Mode()&os.ModeSymlink == 0 {
			return false, fmt.Errorf("conflict: %s exists and is not a symlink", dst)
		}
		cur, _ := os.Readlink(dst)
		if cur == src {
			return false, nil
		}
		if !managed(cur, contentRoot) {
			return false, fmt.Errorf("conflict: %s is a symlink to %s, outside managed content %s; not changing", dst, cur, contentRoot)
		}
		if err := os.Remove(dst); err != nil {
			return false, err
		}
		if err := os.Symlink(src, dst); err != nil {
			return false, err
		}
		return true, nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return false, err
	}
	if err := os.Symlink(src, dst); err != nil {
		return false, err
	}
	return true, nil
}

// Remove deletes dst only when it is a symlink pointing into contentRoot.
// A user's own file or a foreign link is a conflict error — pruning must never
// destroy anything homonto does not own. A missing dst is fine (already gone).
func Remove(dst, contentRoot string) error {
	fi, err := os.Lstat(dst)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("conflict: %s exists and is not a symlink; not removing", dst)
	}
	target, err := os.Readlink(dst)
	if err != nil {
		return err
	}
	if !managed(target, contentRoot) {
		return fmt.Errorf("conflict: %s links to %s, outside managed content %s; not removing", dst, target, contentRoot)
	}
	return os.Remove(dst)
}

// IsManaged reports whether dst is a symlink pointing into contentRoot — a link
// homonto created and may therefore safely relocate or prune. A missing path, a
// real file, or a foreign symlink all return false (leave-it-alone), so callers
// can prune only what is unambiguously theirs without risking a user's file.
func IsManaged(dst, contentRoot string) bool {
	fi, err := os.Lstat(dst)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(dst)
	if err != nil {
		return false
	}
	return managed(target, contentRoot)
}

// Op is a pending link change for dst -> src. Cur is the current symlink
// target, empty when dst does not exist yet (a create).
type Op struct {
	Dst, Src, Cur string
}

// Plan returns the link changes (dst->src) that would be made. Links already
// pointing at src are omitted. A non-symlink at dst is a conflict error, and so
// is a symlink pointing outside contentRoot — a foreign, user-owned link that
// Apply must never repoint. Only a symlink pointing inside contentRoot (one of
// ours) is planned as a relink.
func Plan(srcs map[string]string, contentRoot string) ([]Op, error) {
	var out []Op
	for dst, src := range srcs {
		fi, err := os.Lstat(dst)
		if err != nil {
			out = append(out, Op{Dst: dst, Src: src})
			continue
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			return nil, fmt.Errorf("conflict: %s exists and is not a symlink", dst)
		}
		cur, _ := os.Readlink(dst)
		if cur == src {
			continue
		}
		if !managed(cur, contentRoot) {
			return nil, fmt.Errorf("conflict: %s is a symlink to %s, outside managed content %s; not changing", dst, cur, contentRoot)
		}
		out = append(out, Op{Dst: dst, Src: src, Cur: cur})
	}
	return out, nil
}

// LinkPlan returns descriptions of links (dst->src) that would change.
func LinkPlan(srcs map[string]string, contentRoot string) ([]string, error) {
	ops, err := Plan(srcs, contentRoot)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, op := range ops {
		if op.Cur == "" {
			out = append(out, fmt.Sprintf("+ link %s -> %s", op.Dst, op.Src))
		} else {
			out = append(out, fmt.Sprintf("~ relink %s -> %s", op.Dst, op.Src))
		}
	}
	return out, nil
}
