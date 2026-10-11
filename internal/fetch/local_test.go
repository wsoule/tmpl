package fetch

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, root, rel, body string, mode os.FileMode) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

func TestLocal(t *testing.T) {
	root := t.TempDir()
	write(t, root, "package.json", "{}", 0o644)
	write(t, root, "bin/run", "#!/bin/sh", 0o755)
	write(t, root, "node_modules/x/index.js", "x", 0o644)
	write(t, root, ".git/HEAD", "ref", 0o644)
	if err := os.Symlink("package.json", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/passwd", filepath.Join(root, "bad")); err != nil {
		t.Fatal(err)
	}

	res, err := Local(root)
	if err != nil {
		t.Fatal(err)
	}
	got := paths(res)
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
	if got["bin/run"].Mode != 0o755 {
		t.Errorf("bin/run mode = %v", got["bin/run"].Mode)
	}
	if got["link"].LinkTarget != "package.json" {
		t.Errorf("link = %+v", got["link"])
	}
	if len(res.Warnings) != 1 {
		t.Errorf("warnings = %v", res.Warnings)
	}
}

func TestLocalMissing(t *testing.T) {
	if _, err := Local(filepath.Join(t.TempDir(), "nope")); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v", err)
	}
	empty := t.TempDir()
	if _, err := Local(empty); !errors.Is(err, ErrNotFound) {
		t.Errorf("empty dir err = %v", err)
	}
}
