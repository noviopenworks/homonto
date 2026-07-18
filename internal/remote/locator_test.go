package remote

import "testing"

func TestParseRemoteSource(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantURL   string
		wantKind  TransportKind
		wantError bool
	}{
		{"https tgz", "remote:https://h.test/x.tar.gz", "https://h.test/x.tar.gz", TransportHTTPS, false},
		{"file url", "remote:file:///tmp/x.tar.gz", "file:///tmp/x.tar.gz", TransportFile, false},
		{"git url", "remote:git+https://h.test/r.git#deadbeef", "git+https://h.test/r.git#deadbeef", TransportGit, false},
		{"not remote prefix", "builtin:x", "", "", true},
		{"empty url", "remote:", "", "", true},
		{"plain http rejected", "remote:http://h.test/x.tgz", "", "", true},
		{"git scheme rejected as insecure", "remote:git://h.test/r.git#deadbeef", "", "", true},
		{"unknown scheme", "remote:ftp://h.test/x", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rs, err := ParseRemoteSource(c.in)
			if c.wantError {
				if err == nil {
					t.Fatalf("ParseRemoteSource(%q) expected error, got %+v", c.in, rs)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRemoteSource(%q) unexpected error: %v", c.in, err)
			}
			if rs.URL != c.wantURL {
				t.Errorf("URL = %q, want %q", rs.URL, c.wantURL)
			}
			if rs.Transport != c.wantKind {
				t.Errorf("Transport = %q, want %q", rs.Transport, c.wantKind)
			}
		})
	}
}

func TestIsRemoteSource(t *testing.T) {
	if !IsRemoteSource("remote:https://h.test/x") {
		t.Error("remote: prefix should be recognized")
	}
	if IsRemoteSource("builtin:x") || IsRemoteSource("local:x") {
		t.Error("builtin/local must not be remote")
	}
}
