package remote

import (
	"net/url"
	"strings"
)

// secretQueryKeys are query parameter names commonly used to carry credentials.
// Their values are redacted before a locator is persisted or reported.
var secretQueryKeys = map[string]bool{
	"token":         true,
	"access_token":  true,
	"api_key":       true,
	"apikey":        true,
	"key":           true,
	"secret":        true,
	"password":      true,
	"passwd":        true,
	"pat":           true,
	"private_token": true,
	"auth":          true,
}

// redactedMarker replaces a redacted secret query value.
const redactedMarker = "REDACTED"

// RedactLocator returns a canonical form of a remote locator with any embedded
// credentials removed, so it is safe to persist in remote.lock.json or surface
// in an error or log line. URL userinfo (user:pass@, including a bare token as
// the username) is dropped entirely, and the values of known secret query
// parameters are replaced with a marker. A locator that does not parse as a URL
// falls back to a best-effort userinfo strip so a secret still cannot leak.
func RedactLocator(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return stripUserinfoFallback(raw)
	}
	changed := false
	if u.User != nil {
		// Drop userinfo wholesale: a bare username can itself be a token
		// (e.g. a PAT used as https://<token>@host/...), so keeping the
		// username is not safe.
		u.User = nil
		changed = true
	}
	if u.RawQuery != "" {
		vals := u.Query()
		for k := range vals {
			if secretQueryKeys[strings.ToLower(k)] {
				vals.Set(k, redactedMarker)
				changed = true
			}
		}
		if changed {
			u.RawQuery = vals.Encode()
		}
	}
	if !changed {
		return raw
	}
	return u.String()
}

// stripUserinfoFallback removes a "user:pass@" (or "token@") userinfo segment
// from the authority of a "scheme://…" string without a full URL parse. It only
// touches the segment before the first '/', '?' or '#' after the authority so it
// cannot corrupt a path or query that happens to contain '@'.
func stripUserinfoFallback(raw string) string {
	i := strings.Index(raw, "://")
	if i < 0 {
		return raw
	}
	rest := raw[i+3:]
	// Authority ends at the first '/', '?' or '#'.
	end := len(rest)
	if j := strings.IndexAny(rest, "/?#"); j >= 0 {
		end = j
	}
	authority := rest[:end]
	at := strings.LastIndex(authority, "@")
	if at < 0 {
		return raw
	}
	return raw[:i+3] + authority[at+1:] + rest[end:]
}
