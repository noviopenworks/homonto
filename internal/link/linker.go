package link

import (
	"fmt"
	"os"
	"path/filepath"
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

// LinkPlan returns descriptions of links (dst->src) that would change.
func LinkPlan(srcs map[string]string) ([]string, error) {
	var out []string
	for dst, src := range srcs {
		fi, err := os.Lstat(dst)
		if err != nil {
			out = append(out, fmt.Sprintf("+ link %s -> %s", dst, src))
			continue
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			return nil, fmt.Errorf("conflict: %s exists and is not a symlink", dst)
		}
		if cur, _ := os.Readlink(dst); cur != src {
			out = append(out, fmt.Sprintf("~ relink %s -> %s", dst, src))
		}
	}
	return out, nil
}
