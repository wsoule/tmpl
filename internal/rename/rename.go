// Package rename fills the project name into template files by replacing
// the literal {{name}} token.
package rename

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/wsoule/tmpl/internal/fetch"
)

// Token is the placeholder template authors put where the project name goes.
const Token = "{{name}}"

var valid = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// Validate checks that name is safe to use as a package name.
func Validate(name string) error {
	if !valid.MatchString(name) {
		return fmt.Errorf("invalid project name %q: use lowercase letters, digits, '.', '_' or '-' (try %q)", name, Suggest(name))
	}
	return nil
}

// Suggest turns an arbitrary string, usually a folder name, into a valid
// project name: lowercase, other characters replaced by '-', no leading
// punctuation. Falls back to "app".
func Suggest(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.TrimLeft(b.String(), "._-")
	if out == "" {
		return "app"
	}
	return out
}

// IsText reports whether data looks like text: no NUL byte in the first 8 KB.
func IsText(data []byte) bool {
	return !bytes.Contains(data[:min(len(data), 8192)], []byte{0})
}

// Uses reports whether any text file contains Token.
func Uses(files []fetch.File) bool {
	for _, f := range files {
		if f.LinkTarget == "" && IsText(f.Data) && bytes.Contains(f.Data, []byte(Token)) {
			return true
		}
	}
	return false
}

// Apply replaces Token with name in every text file, in place, and returns
// the paths it changed. Symlinks and binary files are left alone.
func Apply(files []fetch.File, name string) []string {
	var changed []string
	tok, repl := []byte(Token), []byte(name)
	for i := range files {
		f := &files[i]
		if f.LinkTarget != "" || !IsText(f.Data) || !bytes.Contains(f.Data, tok) {
			continue
		}
		f.Data = bytes.ReplaceAll(f.Data, tok, repl)
		changed = append(changed, f.Path)
	}
	return changed
}
