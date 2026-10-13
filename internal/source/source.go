// Package source turns the <source> argument a user types into a concrete
// template location: a local directory or a path inside a GitHub repo.
package source

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type Kind int

const (
	Local Kind = iota
	GitHub
)

type Source struct {
	Kind   Kind
	Path   string // Local: absolute directory
	Owner  string // GitHub
	Repo   string // GitHub
	Subdir string // GitHub: slash-separated path inside the repo, "" for root
	Ref    string // GitHub: branch/tag/sha, "" for the default branch
	Bare   bool   // true when the user typed a bare template name
}

type Options struct {
	Cwd         string // resolves relative local paths
	Home        string // expands "~"
	DefaultRepo string // "owner/repo[/path][@ref]" that bare names resolve into
}

// DefaultRepo is where bare template names live unless TMPL_REPO is set.
const DefaultRepo = "wsoule/tmpl"

var bareName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// RepoFromEnv returns the repo bare names resolve into: TMPL_REPO, or DefaultRepo.
func RepoFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("TMPL_REPO")); v != "" {
		return v
	}
	return DefaultRepo
}

// Parse resolves a source string. Order: local path (only when it starts with
// ".", "~" or "/"), then GitHub path (anything with a "/"), then bare name
// inside o.DefaultRepo's templates/. A folder in the current directory never
// shadows a template name; type "./name" to use it.
func Parse(raw string, o Options) (Source, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Source{}, errors.New("empty template source")
	}
	if p, ok := localPath(s, o); ok {
		if p == string(filepath.Separator) {
			return Source{}, fmt.Errorf("invalid template source %q", raw)
		}
		return Source{Kind: Local, Path: p}, nil
	}
	if strings.Contains(s, "/") {
		return parseGitHub(s)
	}
	if !bareName.MatchString(s) {
		return Source{}, fmt.Errorf("invalid template name %q", s)
	}
	src, err := Templates(o.DefaultRepo)
	if err != nil {
		return Source{}, err
	}
	src.Subdir = path.Join(src.Subdir, s)
	src.Bare = true
	return src, nil
}

// Templates returns the templates/ directory of a default repo string.
func Templates(defaultRepo string) (Source, error) {
	src, err := parseGitHub(defaultRepo)
	if err != nil {
		return Source{}, fmt.Errorf("invalid template repo %q (set TMPL_REPO to owner/repo): %w", defaultRepo, err)
	}
	src.Subdir = path.Join(src.Subdir, "templates")
	return src, nil
}

// localPath reports whether s is written as a local path ("./x", "../x",
// "~/x", "/x"), returning its absolute path. It never checks the disk, so the
// same command means the same thing in every directory.
func localPath(s string, o Options) (string, bool) {
	switch {
	case s == "~" || strings.HasPrefix(s, "~/"):
		return filepath.Join(o.Home, s[1:]), true
	case filepath.IsAbs(s):
		return filepath.Clean(s), true
	case s == "." || s == ".." || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../"):
		return filepath.Join(o.Cwd, s), true
	}
	return "", false
}

// parseGitHub parses "owner/repo[/path][@ref]", with an optional
// "https://github.com/" or "github.com/" prefix. The ref is everything after
// the first "@", so refs may contain slashes.
func parseGitHub(s string) (Source, error) {
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "github.com/")
	ref := ""
	if i := strings.Index(s, "@"); i >= 0 {
		s, ref = s[:i], s[i+1:]
		if ref == "" {
			return Source{}, errors.New("empty ref after @")
		}
	}
	parts := strings.Split(strings.Trim(s, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return Source{}, fmt.Errorf("expected owner/repo[/path][@ref], got %q", s)
	}
	for _, p := range parts[2:] {
		if p == "" || p == "." || p == ".." {
			return Source{}, fmt.Errorf("invalid path in %q", s)
		}
	}
	return Source{
		Kind:   GitHub,
		Owner:  parts[0],
		Repo:   parts[1],
		Subdir: strings.Join(parts[2:], "/"),
		Ref:    ref,
	}, nil
}
