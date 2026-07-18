package remote

import (
	"fmt"
	"strings"
)

// TransportKind identifies how a remote source is fetched.
type TransportKind string

const (
	TransportHTTPS TransportKind = "https"
	TransportFile  TransportKind = "file"
	TransportGit   TransportKind = "git"
)

// RemoteSourcePrefix is the source-string prefix that marks a remote resource.
const RemoteSourcePrefix = "remote:"

// RemoteSource is a parsed remote locator (without its pin, which is a sibling
// config field resolved by the loader).
type RemoteSource struct {
	Raw       string
	URL       string // the locator with the remote: prefix stripped
	Transport TransportKind
}

// IsRemoteSource reports whether a config source string names a remote resource.
func IsRemoteSource(source string) bool {
	return strings.HasPrefix(source, RemoteSourcePrefix)
}

// ParseRemoteSource parses "remote:<url>" and selects a transport by scheme.
// Only https (not plain http), file, and git+https/git+file schemes are
// accepted; git:// and http:// are rejected as insecure, and anything else
// fails closed.
func ParseRemoteSource(source string) (RemoteSource, error) {
	if !IsRemoteSource(source) {
		return RemoteSource{}, fmt.Errorf("source %q is not a remote: source", source)
	}
	url := strings.TrimPrefix(source, RemoteSourcePrefix)
	if url == "" {
		return RemoteSource{}, fmt.Errorf("remote source %q has an empty URL", RedactLocator(source))
	}
	kind, err := transportForURL(url)
	if err != nil {
		return RemoteSource{}, err
	}
	return RemoteSource{Raw: source, URL: url, Transport: kind}, nil
}

func transportForURL(url string) (TransportKind, error) {
	switch {
	case strings.HasPrefix(url, "git+https://") || strings.HasPrefix(url, "git+file://"):
		return TransportGit, nil
	case strings.HasPrefix(url, "https://"):
		return TransportHTTPS, nil
	case strings.HasPrefix(url, "file://"):
		return TransportFile, nil
	case strings.HasPrefix(url, "git://") || strings.HasPrefix(url, "http://"):
		// git:// is plaintext/unauthenticated and http:// is cleartext; both are
		// MITM-able and incompatible with the digest-pinned trust boundary the
		// rest of this package enforces. Require TLS or a local file transport.
		return "", fmt.Errorf("remote source %q: insecure scheme (git:// and http:// are not allowed; use https://, file://, or git+https://)", RedactLocator(url))
	default:
		return "", fmt.Errorf("remote source %q: unsupported scheme (want https://, file://, or git+https://)", RedactLocator(url))
	}
}
