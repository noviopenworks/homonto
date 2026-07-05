package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/tidwall/gjson"
)

var (
	secretValueLike = regexp.MustCompile(`^(sk-|github_pat_|ghp_|xox|glpat-|npm_|AIza|Bearer )`)
	secretKeyLike   = regexp.MustCompile(`(_KEY|_TOKEN|_SECRET|_PASSWORD|_CREDENTIALS)$|^DATABASE_URL$`)
)

// Import reads existing tool config into a homonto Config, redacting any value
// that looks like a literal secret into a ${pass:...} reference. Returns the
// config and warnings naming each redaction.
func Import(home string) (*config.Config, []string, error) {
	c := &config.Config{MCPs: map[string]config.MCP{}}
	var warnings []string

	path := filepath.Join(home, ".claude.json")
	mj, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		warnings = append(warnings, fmt.Sprintf("skipped %s: %v", path, err))
	}
	if err == nil {
		doc, _ := jsonutil.Standardize(mj)
		gjson.GetBytes(doc, "mcpServers").ForEach(func(name, server gjson.Result) bool {
			var cmd []string
			// Real Claude Code schema: command is a string, args a separate
			// array. Legacy homonto exports used a single command array.
			if command := server.Get("command"); command.Type == gjson.String {
				cmd = append(cmd, command.String())
				for _, v := range server.Get("args").Array() {
					cmd = append(cmd, v.String())
				}
			} else {
				for _, v := range command.Array() {
					cmd = append(cmd, v.String())
				}
			}
			env := map[string]string{}
			server.Get("env").ForEach(func(k, v gjson.Result) bool {
				val := v.String()
				if redacted, hit := redact(name.String(), k.String(), val); hit {
					warnings = append(warnings, fmt.Sprintf("redacted %s.%s -> %s", name.String(), k.String(), redacted))
					val = redacted
				}
				env[k.String()] = val
				return true
			})
			m := config.MCP{Command: cmd, Targets: []string{"claude"}}
			if len(env) > 0 {
				m.Env = env
			}
			c.MCPs[name.String()] = m
			return true
		})
	}
	return c, warnings, nil
}

func redact(server, key, val string) (string, bool) {
	if strings.HasPrefix(val, "${") {
		return val, false
	}
	if secretValueLike.MatchString(val) || secretKeyLike.MatchString(key) {
		return fmt.Sprintf("${pass:imported/%s/%s}", server, key), true
	}
	return val, false
}

// MarshalTOML serializes a Config to TOML.
func MarshalTOML(c *config.Config) ([]byte, error) { return toml.Marshal(c) }
