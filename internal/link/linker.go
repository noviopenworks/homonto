package link

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Link ensures dst is a symlink to src, returning whether it changed. It never
// clobbers: if dst exists and is not our symlink, it returns a "conflict" error.
func Link(src, dst string) (bool, error) {
	if fi, err := os.Lstat(dst); err == nil {
		if fi.Mode()&os.ModeSymlink == 0 {
			return false, fmt.Errorf("conflict: %s exists and is not a symlink", dst)
		}
		cur, _ := os.Readlink(dst)
		if cur == src {
			return false, nil
		}
		return false, fmt.Errorf("conflict: %s links to %s, not %s", dst, cur, src)
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
	if !strings.HasPrefix(target, contentRoot+string(os.PathSeparator)) {
		return fmt.Errorf("conflict: %s links to %s, outside managed content %s; not removing", dst, target, contentRoot)
	}
	return os.Remove(dst)
}

// Op is a pending link change for dst -> src. Cur is the current symlink
// target, empty when dst does not exist yet (a create).
type Op struct {
	Dst, Src, Cur string
}

// Plan returns the link changes (dst->src) that would be made. Links already
// pointing at src are omitted; a non-symlink at dst is a conflict error.
func Plan(srcs map[string]string) ([]Op, error) {
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
		if cur, _ := os.Readlink(dst); cur != src {
			out = append(out, Op{Dst: dst, Src: src, Cur: cur})
		}
	}
	return out, nil
}

// LinkPlan returns descriptions of links (dst->src) that would change.
func LinkPlan(srcs map[string]string) ([]string, error) {
	ops, err := Plan(srcs)
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
