package catalog

import (
	"fmt"
	"strconv"
	"strings"
)

// parseVer parses a plain three-part "x.y.z" version into its numeric
// components. Framework versions are plain semver-shaped strings (no pre-release
// or build metadata), so anything else is an error rather than a silent pass.
func parseVer(s string) ([3]int, error) {
	var v [3]int
	parts := strings.Split(strings.TrimSpace(s), ".")
	if len(parts) != 3 {
		return v, fmt.Errorf("version %q is not x.y.z", s)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, fmt.Errorf("version %q has a non-numeric component %q", s, p)
		}
		v[i] = n
	}
	return v, nil
}

// cmpVer returns -1, 0, or 1 comparing a to b component-wise (numeric).
func cmpVer(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		switch {
		case a[i] < b[i]:
			return -1
		case a[i] > b[i]:
			return 1
		}
	}
	return 0
}

// satisfies reports whether version v meets constraint c. c is one of
// ">=x.y.z", ">x.y.z", "<=x.y.z", "<x.y.z", "=x.y.z", or a bare "x.y.z" (exact).
// A malformed version/constraint or an unsupported operator is an error, so an
// unrecognized constraint fails loud rather than silently passing.
func satisfies(v, c string) (bool, error) {
	vv, err := parseVer(v)
	if err != nil {
		return false, err
	}
	c = strings.TrimSpace(c)
	op := "="
	rest := c
	for _, cand := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(c, cand) {
			op = cand
			rest = strings.TrimSpace(c[len(cand):])
			break
		}
	}
	cv, err := parseVer(rest)
	if err != nil {
		return false, fmt.Errorf("constraint %q: %w", c, err)
	}
	d := cmpVer(vv, cv)
	switch op {
	case ">=":
		return d >= 0, nil
	case ">":
		return d > 0, nil
	case "<=":
		return d <= 0, nil
	case "<":
		return d < 0, nil
	case "=":
		return d == 0, nil
	default:
		return false, fmt.Errorf("constraint %q: unsupported operator", c)
	}
}

// parseDep splits a dependency entry "name" or "name@constraint" into the
// framework name and its version constraint ("" when unconstrained).
func parseDep(entry string) (name, constraint string) {
	if i := strings.LastIndex(entry, "@"); i >= 0 {
		return entry[:i], entry[i+1:]
	}
	return entry, ""
}

// parseCapability splits a capability "name@major" into its name and integer
// major version. The name must be non-empty and major a non-negative integer;
// anything else is an error so a malformed capability fails loud.
func parseCapability(s string) (name string, major int, err error) {
	i := strings.LastIndex(s, "@")
	if i <= 0 || i == len(s)-1 {
		return "", 0, fmt.Errorf("capability %q is not name@major", s)
	}
	name = s[:i]
	n, e := strconv.Atoi(s[i+1:])
	if e != nil || n < 0 {
		return "", 0, fmt.Errorf("capability %q has a non-integer major", s)
	}
	return name, n, nil
}
