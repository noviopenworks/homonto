package catalog

import (
	"io/fs"
	"testing"

	embedded "github.com/noviopenworks/homonto/catalog"
)

func TestSubagentsEmbedded(t *testing.T) {
	for _, name := range []string{"onto-reviewer", "onto-explorer", "comet-navigator"} {
		p := "subagents/" + name + ".md"
		if _, err := fs.Stat(embedded.FS, p); err != nil {
			t.Errorf("%s not embedded: %v", p, err)
		}
	}
}
