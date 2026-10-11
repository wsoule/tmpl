// Package catalog turns the contents of a templates/ directory into the
// list `tmpl list` prints.
package catalog

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/wsoule/tmpl/internal/fetch"
)

// Entry is one template: its directory name and optional description.
type Entry struct {
	Name        string
	Description string
}

// List returns one entry per top-level directory in files, sorted by name.
// Descriptions come from each template's tmpl.json; unreadable manifests are ignored.
func List(files []fetch.File) []Entry {
	desc := map[string]string{}
	seen := map[string]bool{}
	for _, f := range files {
		i := strings.Index(f.Path, "/")
		if i < 0 {
			continue
		}
		name := f.Path[:i]
		seen[name] = true
		if f.Path[i+1:] == "tmpl.json" {
			var m struct {
				Description string `json:"description"`
			}
			if json.Unmarshal(f.Data, &m) == nil {
				desc[name] = m.Description
			}
		}
	}
	entries := make([]Entry, 0, len(seen))
	for name := range seen {
		entries = append(entries, Entry{Name: name, Description: desc[name]})
	}
	sort.Slice(entries, func(a, b int) bool { return entries[a].Name < entries[b].Name })
	return entries
}
