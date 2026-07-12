package remote

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	maxRedirects = 5
	httpTimeout  = 60 * time.Second
)

// Fetch retrieves a remote source into a validated Tree, selecting a transport
// by scheme. It never writes to the cache or any target; verification (pin match,
// revocation) is the caller's responsibility. The returned size is the fetched
// byte count (compressed download or on-disk archive size).
func Fetch(ctx context.Context, src RemoteSource, lim Limits) (Tree, int64, error) {
	switch src.Transport {
	case TransportHTTPS:
		return fetchHTTPS(ctx, src.URL, lim, nil)
	case TransportFile:
		return fetchFile(ctx, src.URL, lim)
	case TransportGit:
		return fetchGit(ctx, src.URL, lim)
	default:
		return Tree{}, 0, fmt.Errorf("remote: unsupported transport %q", src.Transport)
	}
}

// fetchHTTPS downloads an https tar.gz with a redirect cap, timeout, and a size
// ceiling, then validates it. The client is injectable for tests; a nil client
// uses a default. Only https is reachable here (the locator rejects plain http).
func fetchHTTPS(ctx context.Context, url string, lim Limits, client *http.Client) (Tree, int64, error) {
	if !strings.HasPrefix(url, "https://") {
		return Tree{}, 0, fmt.Errorf("remote: https transport requires an https:// URL, got %q", url)
	}
	base := http.DefaultClient
	if client != nil {
		base = client
	}
	c := *base // shallow copy so we can set redirect/timeout without mutating the caller's client
	c.Timeout = httpTimeout
	c.CheckRedirect = func(_ *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return fmt.Errorf("remote: stopped after %d redirects", maxRedirects)
		}
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Tree{}, 0, fmt.Errorf("remote: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return Tree{}, 0, fmt.Errorf("remote: fetch %q: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Tree{}, 0, fmt.Errorf("remote: fetch %q: unexpected status %s", url, resp.Status)
	}
	// Bound the compressed download; the decompressed stream is bounded again by
	// ValidateTarGz. One extra byte detects an overflow.
	compressed, err := io.ReadAll(io.LimitReader(resp.Body, lim.MaxTotalBytes+1))
	if err != nil {
		return Tree{}, 0, fmt.Errorf("remote: reading body: %w", err)
	}
	if int64(len(compressed)) > lim.MaxTotalBytes {
		return Tree{}, 0, fmt.Errorf("remote: download exceeds the %d-byte cap", lim.MaxTotalBytes)
	}
	tree, err := ValidateTarGz(bytes.NewReader(compressed), lim)
	if err != nil {
		return Tree{}, 0, err
	}
	return tree, int64(len(compressed)), nil
}

// fetchFile reads a local file:// source: either a .tar.gz archive or a
// directory, both run through the same validation.
func fetchFile(_ context.Context, url string, lim Limits) (Tree, int64, error) {
	p := strings.TrimPrefix(url, "file://")
	if p == "" {
		return Tree{}, 0, fmt.Errorf("remote: file source has an empty path")
	}
	info, err := os.Stat(p)
	if err != nil {
		return Tree{}, 0, fmt.Errorf("remote: file source: %w", err)
	}
	if info.IsDir() {
		tree, size, err := treeFromDir(p, lim, nil)
		if err != nil {
			return Tree{}, 0, err
		}
		return tree, size, nil
	}
	f, err := os.Open(p)
	if err != nil {
		return Tree{}, 0, fmt.Errorf("remote: file source: %w", err)
	}
	defer f.Close()
	tree, err := ValidateTarGz(f, lim)
	if err != nil {
		return Tree{}, 0, err
	}
	return tree, info.Size(), nil
}

// fetchGit shallow-clones a pinned ref into a temp worktree and validates its
// tree (excluding .git). Trust is governed by the content pin, so a moved tag or
// branch is caught by the digest at verify time.
func fetchGit(ctx context.Context, url string, lim Limits) (Tree, int64, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return Tree{}, 0, fmt.Errorf("remote: git transport requires git on PATH: %w", err)
	}
	cloneURL := strings.TrimPrefix(url, "git+")
	ref := ""
	if i := strings.LastIndex(cloneURL, "#"); i >= 0 {
		ref = cloneURL[i+1:]
		cloneURL = cloneURL[:i]
	}
	if ref == "" {
		return Tree{}, 0, fmt.Errorf("remote: git source %q must pin a ref with #<commit-or-tag>", url)
	}
	tmp, err := os.MkdirTemp("", "homonto-git-*")
	if err != nil {
		return Tree{}, 0, fmt.Errorf("remote: %w", err)
	}
	defer os.RemoveAll(tmp)

	clone := exec.CommandContext(ctx, "git", "-c", "protocol.file.allow=always", "clone", "--quiet", "--no-checkout", cloneURL, tmp)
	if out, err := clone.CombinedOutput(); err != nil {
		return Tree{}, 0, fmt.Errorf("remote: git clone failed: %v: %s", err, out)
	}
	checkout := exec.CommandContext(ctx, "git", "-C", tmp, "-c", "advice.detachedHead=false", "checkout", "--quiet", ref)
	if out, err := checkout.CombinedOutput(); err != nil {
		return Tree{}, 0, fmt.Errorf("remote: git checkout %q failed: %v: %s", ref, err, out)
	}
	if err := os.RemoveAll(filepath.Join(tmp, ".git")); err != nil {
		return Tree{}, 0, fmt.Errorf("remote: %w", err)
	}
	return treeFromDir(tmp, lim, nil)
}

// treeFromDir walks a directory into a validated Tree, rejecting symlinks and
// enforcing the same caps as archive extraction. skip, if non-nil, drops a
// relative path from the tree.
func treeFromDir(root string, lim Limits, skip func(rel string) bool) (Tree, int64, error) {
	var (
		files []FileEntry
		total int64
		count int
	)
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		if skip != nil && skip(rel) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		// WalkDir uses Lstat, so a symlink shows its own type here.
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("remote: source directory contains a symlink %q", rel)
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("remote: source directory contains a non-regular file %q", rel)
		}
		count++
		if count > lim.MaxEntries {
			return fmt.Errorf("remote: source exceeds %d entries", lim.MaxEntries)
		}
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		if info.Size() > lim.MaxEntryBytes {
			return fmt.Errorf("remote: source file %q exceeds the %d-byte per-entry cap", rel, lim.MaxEntryBytes)
		}
		total += info.Size()
		if total > lim.MaxTotalBytes {
			return fmt.Errorf("remote: source exceeds the %d-byte total cap", lim.MaxTotalBytes)
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		files = append(files, FileEntry{Path: rel, Mode: uint32(info.Mode()) & 0o777, Data: data})
		return nil
	})
	if err != nil {
		return Tree{}, 0, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return Tree{Files: files}, total, nil
}
