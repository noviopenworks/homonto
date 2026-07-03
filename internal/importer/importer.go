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

var secretLike = regexp.MustCompile(`^(sk-|github_pat_|ghp_|xox)`)

// Import reads existing tool config into a homonto Config, redacting any value
// that looks like a literal secret into a ${pass:...} reference. Returns the
// config and warnings naming each redaction.
func Import(home string) (*config.Config, []string, error) {
	c := &config.Config{MCPs: map[string]config.MCP{}}
	var warnings []string

	mj, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err == nil {
		doc, _ := jsonutil.Standardize(mj)
		gjson.GetBytes(doc, "mcpServers").ForEach(func(name, server gjson.Result) bool {
			var cmd []string
			for _, v := range server.Get("command").Array() {
				cmd = append(cmd, v.String())
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
	if secretLike.MatchString(val) || strings.HasSuffix(key, "_KEY") || strings.HasSuffix(key, "_TOKEN") {
		return fmt.Sprintf("${pass:imported/%s/%s}", server, key), true
	}
	return val, false
}

// MarshalTOML serializes a Config to TOML.
func MarshalTOML(c *config.Config) ([]byte, error) { return toml.Marshal(c) }
