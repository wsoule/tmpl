package source

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cwd := t.TempDir()
	if err := os.Mkdir(filepath.Join(cwd, "localdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	o := Options{Cwd: cwd, Home: "/home/me", DefaultRepo: "wsoule/tmpl"}

	tests := []struct {
		in   string
		want Source
	}{
		{"pierre", Source{Kind: GitHub, Owner: "wsoule", Repo: "tmpl", Subdir: "templates/pierre", Bare: true}},
		{"alice/templates", Source{Kind: GitHub, Owner: "alice", Repo: "templates"}},
		{"alice/templates/web/next", Source{Kind: GitHub, Owner: "alice", Repo: "templates", Subdir: "web/next"}},
		{"alice/templates/web@v2", Source{Kind: GitHub, Owner: "alice", Repo: "templates", Subdir: "web", Ref: "v2"}},
		{"alice/templates/web@feature/x", Source{Kind: GitHub, Owner: "alice", Repo: "templates", Subdir: "web", Ref: "feature/x"}},
		{"https://github.com/alice/templates", Source{Kind: GitHub, Owner: "alice", Repo: "templates"}},
		{"github.com/alice/templates/web", Source{Kind: GitHub, Owner: "alice", Repo: "templates", Subdir: "web"}},
		{"./tpl", Source{Kind: Local, Path: filepath.Join(cwd, "tpl")}},
		{"../tpl", Source{Kind: Local, Path: filepath.Join(filepath.Dir(cwd), "tpl")}},
		{"~/tpl", Source{Kind: Local, Path: "/home/me/tpl"}},
		{"/abs/tpl", Source{Kind: Local, Path: "/abs/tpl"}},
		{"localdir", Source{Kind: Local, Path: filepath.Join(cwd, "localdir")}},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in, o)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.in, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Parse(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestParseDefaultRepoOverride(t *testing.T) {
	got, err := Parse("web", Options{Cwd: t.TempDir(), DefaultRepo: "alice/stuff/starters@main"})
	if err != nil {
		t.Fatal(err)
	}
	want := Source{Kind: GitHub, Owner: "alice", Repo: "stuff", Subdir: "starters/templates/web", Ref: "main", Bare: true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseErrors(t *testing.T) {
	o := Options{Cwd: t.TempDir(), DefaultRepo: "wsoule/tmpl"}
	for _, in := range []string{"", "  ", "alice/", "/", "alice/repo@", "alice/repo/../x", "bad name!"} {
		if _, err := Parse(in, o); err == nil {
			t.Errorf("Parse(%q) expected error", in)
		}
	}
	if _, err := Parse("pierre", Options{Cwd: t.TempDir(), DefaultRepo: "nope"}); err == nil {
		t.Error("expected error for invalid default repo")
	}
}

func TestRepoFromEnv(t *testing.T) {
	t.Setenv("TMPL_REPO", "")
	if got := RepoFromEnv(); got != DefaultRepo {
		t.Errorf("got %q", got)
	}
	t.Setenv("TMPL_REPO", " alice/templates ")
	if got := RepoFromEnv(); got != "alice/templates" {
		t.Errorf("got %q", got)
	}
}

func TestTemplates(t *testing.T) {
	got, err := Templates("wsoule/tmpl")
	if err != nil {
		t.Fatal(err)
	}
	want := Source{Kind: GitHub, Owner: "wsoule", Repo: "tmpl", Subdir: "templates"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
