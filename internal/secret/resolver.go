package secret

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var refRe = regexp.MustCompile(`\$\{([^}]+)\}`)

// Resolver replaces ${...} references with values from pass or the environment.
type Resolver struct {
	Getenv func(string) string
	Pass   func(path string) (string, error)
}

// NewResolver returns a Resolver backed by os.Getenv and `pass show`.
func NewResolver() *Resolver {
	return &Resolver{
		Getenv: os.Getenv,
		Pass: func(path string) (string, error) {
			out, err := exec.Command("pass", "show", path).Output()
			if err != nil {
				return "", fmt.Errorf("pass show %s: %w", path, err)
			}
			return strings.SplitN(strings.TrimRight(string(out), "\n"), "\n", 2)[0], nil
		},
	}
}

// ContainsRef reports whether s contains a ${...} reference.
func ContainsRef(s string) bool { return refRe.MatchString(s) }

// Resolve replaces every ${...} token in s with its resolved value.
func (r *Resolver) Resolve(s string) (string, error) {
	var firstErr error
	out := refRe.ReplaceAllStringFunc(s, func(tok string) string {
		inner := tok[2 : len(tok)-1] // strip ${ }
		var val string
		var err error
		if strings.HasPrefix(inner, "pass:") {
			val, err = r.Pass(strings.TrimPrefix(inner, "pass:"))
		} else {
			val = r.Getenv(inner)
			if val == "" {
				err = fmt.Errorf("env var %s is not set", inner)
			}
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
		return val
	})
	if firstErr != nil {
		return "", firstErr
	}
	return out, nil
}
