// Package fetch loads a template's files into memory from a local directory
// or a GitHub tarball, dropping anything unsafe or never-copied on the way.
package fetch

import (
	"errors"
	"io/fs"
	"path"
	"strings"
)

// File is one template file held in memory.
type File struct {
	Path       string      // slash-separated, relative to the template root
	Mode       fs.FileMode // permission bits only
	Data       []byte      // nil for symlinks
	LinkTarget string      // non-empty for symlinks
}

// Result is a fetched template plus non-fatal problems (skipped entries).
type Result struct {
	Files    []File
	Warnings []string
}

// ErrNotFound means the source exists nowhere or contains no files.
var ErrNotFound = errors.New("template not found")

// Ignored reports whether a path must never be copied into a project:
// anything inside .git or node_modules, and macOS .DS_Store files.
func Ignored(rel string) bool {
	for _, seg := range strings.Split(rel, "/") {
		if seg == ".git" || seg == "node_modules" || seg == ".DS_Store" {
			return true
		}
	}
	return false
}

// cleanRel normalizes a slash-separated relative path and rejects paths that
// are empty, absolute, or climb out of the template root with "..".
func cleanRel(p string) (string, bool) {
	if p == "" || path.IsAbs(p) {
		return "", false
	}
	c := path.Clean(p)
	if c == "." || c == ".." || strings.HasPrefix(c, "../") {
		return "", false
	}
	return c, true
}

// linkAncestor returns the symlink in links that is an ancestor of rel, or "".
// A path under a symlink would be written wherever that link points, which a
// lexical check on the path cannot see, so such entries are never kept.
// Keys in links are lowercased, because macOS filesystems treat "A/b" and
// "a/b" as the same path; the writer re-checks against the real filesystem.
func linkAncestor(rel string, links map[string]bool) string {
	for i := 0; i < len(rel); i++ {
		if rel[i] == '/' && links[strings.ToLower(rel[:i])] {
			return rel[:i]
		}
	}
	return ""
}

// linkStaysInside reports whether a symlink stored at rel and pointing at
// target resolves to somewhere inside the template root.
func linkStaysInside(rel, target string) bool {
	if path.IsAbs(target) {
		return false
	}
	_, ok := cleanRel(path.Join(path.Dir(rel), target))
	return ok
}
