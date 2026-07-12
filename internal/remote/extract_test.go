package remote

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"
)

// tarEntry describes one member to write into a test archive.
type tarEntry struct {
	name     string
	typeflag byte
	data     string
	linkname string
}

func buildTar(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		tf := e.typeflag
		if tf == 0 {
			tf = tar.TypeReg
		}
		hdr := &tar.Header{
			Name:     e.name,
			Typeflag: tf,
			Mode:     0o644,
			Size:     int64(len(e.data)),
			Linkname: e.linkname,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if len(e.data) > 0 {
			if _, err := tw.Write([]byte(e.data)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func gz(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestValidateTarHappyPath(t *testing.T) {
	raw := buildTar(t, []tarEntry{
		{name: "b.txt", data: "bbb"},
		{name: "sub/a.txt", data: "aaa"},
		{name: "sub/", typeflag: tar.TypeDir},
	})
	tree, err := ValidateTar(bytes.NewReader(raw), DefaultLimits)
	if err != nil {
		t.Fatalf("valid archive rejected: %v", err)
	}
	if len(tree.Files) != 2 {
		t.Fatalf("want 2 regular files, got %d", len(tree.Files))
	}
	// files sorted by path
	if tree.Files[0].Path != "b.txt" || tree.Files[1].Path != "sub/a.txt" {
		t.Fatalf("files not sorted canonically: %+v", tree.Files)
	}
	if string(tree.Files[1].Data) != "aaa" {
		t.Fatalf("content mismatch: %q", tree.Files[1].Data)
	}
}

func TestValidateTarFailsClosed(t *testing.T) {
	small := Limits{MaxEntries: 5, MaxEntryBytes: 16, MaxTotalBytes: 32}
	cases := []struct {
		name    string
		entries []tarEntry
		lim     Limits
	}{
		{"absolute path", []tarEntry{{name: "/etc/passwd", data: "x"}}, DefaultLimits},
		{"parent traversal", []tarEntry{{name: "../escape", data: "x"}}, DefaultLimits},
		{"nested traversal", []tarEntry{{name: "a/../../escape", data: "x"}}, DefaultLimits},
		{"symlink", []tarEntry{{name: "l", typeflag: tar.TypeSymlink, linkname: "/etc/passwd"}}, DefaultLimits},
		{"hardlink", []tarEntry{{name: "l", typeflag: tar.TypeLink, linkname: "b.txt"}}, DefaultLimits},
		{"char device", []tarEntry{{name: "d", typeflag: tar.TypeChar}}, DefaultLimits},
		{"fifo", []tarEntry{{name: "p", typeflag: tar.TypeFifo}}, DefaultLimits},
		{"per-entry over", []tarEntry{{name: "big", data: "0123456789abcdefXY"}}, small},
		{"total over", []tarEntry{{name: "a", data: "0123456789abcde"}, {name: "b", data: "0123456789abcde"}, {name: "c", data: "0123456789abcde"}}, small},
		{"too many entries", []tarEntry{{name: "1", data: "a"}, {name: "2", data: "a"}, {name: "3", data: "a"}, {name: "4", data: "a"}, {name: "5", data: "a"}, {name: "6", data: "a"}}, small},
		{"duplicate path", []tarEntry{{name: "dup", data: "a"}, {name: "dup", data: "b"}}, DefaultLimits},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			raw := buildTar(t, c.entries)
			_, err := ValidateTar(bytes.NewReader(raw), c.lim)
			if err == nil {
				t.Fatalf("%s: expected fail-closed error, got nil", c.name)
			}
		})
	}
}

func TestValidateTarGz(t *testing.T) {
	raw := gz(t, buildTar(t, []tarEntry{{name: "a.txt", data: "hi"}}))
	tree, err := ValidateTarGz(bytes.NewReader(raw), DefaultLimits)
	if err != nil {
		t.Fatalf("valid gz archive rejected: %v", err)
	}
	if len(tree.Files) != 1 || string(tree.Files[0].Data) != "hi" {
		t.Fatalf("bad tree: %+v", tree.Files)
	}
}

func TestValidateTarGzBombBoundedByTotal(t *testing.T) {
	// A single entry whose declared+written size exceeds the total cap must be
	// rejected while streaming, not after full decompression.
	small := Limits{MaxEntries: 5, MaxEntryBytes: 1024, MaxTotalBytes: 64}
	big := make([]byte, 4096)
	raw := gz(t, buildTar(t, []tarEntry{{name: "bomb", data: string(big)}}))
	if _, err := ValidateTarGz(bytes.NewReader(raw), small); err == nil {
		t.Fatal("gzip bomb should be rejected by the total-size cap")
	}
}
