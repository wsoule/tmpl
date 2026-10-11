package write

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wsoule/tmpl/internal/fetch"
)

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func sample() []fetch.File {
	return []fetch.File{
		{Path: "package.json", Mode: 0o644, Data: []byte("new")},
		{Path: "scripts/run.sh", Mode: 0o755, Data: []byte("#!/bin/sh")},
		{Path: "tmpl.json", Mode: 0o644, Data: []byte(`{"description":"x"}`)},
		{Path: "docs/tmpl.json", Mode: 0o644, Data: []byte("nested is fine")},
		{Path: "link", Mode: 0o777, LinkTarget: "package.json"},
	}
}

func TestFilesIntoNewDir(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "new", "app")
	rep, err := Files(dest, sample(), false)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"package.json", "scripts/run.sh", "docs/tmpl.json", "link"}; !reflect.DeepEqual(rep.Written, want) {
		t.Errorf("written = %v", rep.Written)
	}
	if _, err := os.Stat(filepath.Join(dest, "tmpl.json")); !os.IsNotExist(err) {
		t.Error("root tmpl.json was written")
	}
	fi, err := os.Stat(filepath.Join(dest, "scripts/run.sh"))
	if err != nil || fi.Mode().Perm() != 0o755 {
		t.Errorf("run.sh mode = %v, err = %v", fi.Mode(), err)
	}
	if target, _ := os.Readlink(filepath.Join(dest, "link")); target != "package.json" {
		t.Errorf("link target = %q", target)
	}
}

func TestFilesKeepsExisting(t *testing.T) {
	dest := t.TempDir()
	if err := os.WriteFile(filepath.Join(dest, "package.json"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "Spec.md"), []byte("spec"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := Files(dest, sample(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rep.Kept, []string{"package.json"}) {
		t.Errorf("kept = %v", rep.Kept)
	}
	if read(t, filepath.Join(dest, "package.json")) != "mine" {
		t.Error("existing file was overwritten")
	}
	if read(t, filepath.Join(dest, "Spec.md")) != "spec" {
		t.Error("unrelated file changed")
	}
}

func TestFilesForceOverwrites(t *testing.T) {
	dest := t.TempDir()
	if err := os.WriteFile(filepath.Join(dest, "package.json"), []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("elsewhere", filepath.Join(dest, "link")); err != nil {
		t.Fatal(err)
	}
	rep, err := Files(dest, sample(), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Kept) != 0 {
		t.Errorf("kept = %v", rep.Kept)
	}
	if read(t, filepath.Join(dest, "package.json")) != "new" {
		t.Error("not overwritten")
	}
	if fi, _ := os.Stat(filepath.Join(dest, "package.json")); fi.Mode().Perm() != 0o644 {
		t.Errorf("mode = %v", fi.Mode())
	}
	if target, _ := os.Readlink(filepath.Join(dest, "link")); target != "package.json" {
		t.Errorf("link target = %q", target)
	}
}

// Each link below passes the lexical check in fetch on its own, but together
// they climb out of dest: a/b -> .. is dest itself, so a/b/c -> .. is dest's
// parent, and a/b/c/evil would land next to dest.
func TestFilesRefusesSymlinkChainEscape(t *testing.T) {
	parent := t.TempDir()
	dest := filepath.Join(parent, "app")
	files := []fetch.File{
		{Path: "a/b", Mode: 0o777, LinkTarget: ".."},
		{Path: "a/b/c", Mode: 0o777, LinkTarget: ".."},
		{Path: "a/b/c/evil", Mode: 0o644, Data: []byte("escaped")},
	}
	if _, err := Files(dest, files, false); err == nil {
		t.Error("expected an error for a path escaping dest")
	}
	if _, err := os.Lstat(filepath.Join(parent, "evil")); !os.IsNotExist(err) {
		t.Fatal("file was written outside dest")
	}
}

// A pre-existing symlink in dest must not let template files land outside it.
func TestFilesRefusesExistingSymlinkEscape(t *testing.T) {
	parent := t.TempDir()
	outside := filepath.Join(parent, "outside")
	dest := filepath.Join(parent, "app")
	for _, d := range []string{outside, dest} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(dest, "src")); err != nil {
		t.Fatal(err)
	}
	files := []fetch.File{{Path: "src/index.ts", Mode: 0o644, Data: []byte("x")}}
	if _, err := Files(dest, files, false); err == nil {
		t.Error("expected an error writing through a symlink that leaves dest")
	}
	if _, err := os.Lstat(filepath.Join(outside, "index.ts")); !os.IsNotExist(err) {
		t.Fatal("file was written outside dest")
	}
}
