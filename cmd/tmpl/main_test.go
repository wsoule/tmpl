package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture creates a local template with a {{name}} token, a manifest, an
// executable script and a setup skill.
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"package.json":                  `{"name":"{{name}}"}`,
		"tmpl.json":                     `{"description":"fixture"}`,
		"scripts/dev.sh":                "#!/bin/sh\necho dev",
		".agents/skills/setup/SKILL.md": "setup steps",
		"node_modules/junk/index.js":    "junk",
	}
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(filepath.Join(root, "scripts/dev.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

// nonTTY returns the read end of an already-closed pipe to use as stdin, so
// run() takes the non-interactive path (a pipe is never a terminal).
func nonTTY(t *testing.T) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	t.Cleanup(func() { r.Close() })
	return r
}

func runTmpl(t *testing.T, argv ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(argv, nonTTY(t), &out, &errb)
	return code, out.String(), errb.String()
}

func TestCopyIntoNewFolderNamesProject(t *testing.T) {
	tpl := fixture(t)
	dest := filepath.Join(t.TempDir(), "My App")
	code, out, errOut := runTmpl(t, tpl, dest)
	if code != 0 {
		t.Fatalf("exit %d: %s %s", code, out, errOut)
	}
	b, err := os.ReadFile(filepath.Join(dest, "package.json"))
	if err != nil || string(b) != `{"name":"my-app"}` {
		t.Errorf("package.json = %q, %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "tmpl.json")); !os.IsNotExist(err) {
		t.Error("tmpl.json copied")
	}
	if _, err := os.Stat(filepath.Join(dest, "node_modules")); !os.IsNotExist(err) {
		t.Error("node_modules copied")
	}
	if fi, _ := os.Stat(filepath.Join(dest, "scripts/dev.sh")); fi.Mode().Perm() != 0o755 {
		t.Error("exec bit lost")
	}
	if !strings.Contains(out, "Next: cd ") {
		t.Errorf("no setup hint in non-interactive run: %q", out)
	}
}

func TestCopyIntoNonEmptyFolderKeepsFiles(t *testing.T) {
	tpl := fixture(t)
	dest := filepath.Join(t.TempDir(), "Dogear")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	spec := filepath.Join(dest, "Awesome Feed Build Spec.md")
	if err := os.WriteFile(spec, []byte("spec"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "package.json"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runTmpl(t, tpl, dest, "--no-setup")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if b, _ := os.ReadFile(filepath.Join(dest, "package.json")); string(b) != "mine" {
		t.Error("existing package.json overwritten")
	}
	if b, _ := os.ReadFile(spec); string(b) != "spec" {
		t.Error("spec changed")
	}
	if !strings.Contains(out, "package.json") || !strings.Contains(out, "--force") {
		t.Errorf("kept files not reported: %q", out)
	}
	if strings.Contains(out, "Next:") {
		t.Error("setup hint printed despite --no-setup")
	}
}

func TestExplicitNameIsValidated(t *testing.T) {
	tpl := fixture(t)
	dest := filepath.Join(t.TempDir(), "x")
	code, _, errOut := runTmpl(t, tpl, dest, "--name", "Bad Name")
	if code != 1 || !strings.Contains(errOut, "bad-name") {
		t.Errorf("exit %d, stderr %q", code, errOut)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Error("dest created despite invalid name")
	}
}

func TestMissingTemplateExits2WithoutCreatingDest(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "x")
	code, _, errOut := runTmpl(t, "./does-not-exist", dest)
	if code != 2 || !strings.Contains(errOut, `template "./does-not-exist" not found`) {
		t.Errorf("exit %d, stderr %q", code, errOut)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Error("dest was created")
	}
}

func TestUsageAndVersion(t *testing.T) {
	if code, _, errOut := runTmpl(t); code != 1 || !strings.Contains(errOut, "usage:") {
		t.Errorf("no args: exit %d %q", code, errOut)
	}
	if code, out, _ := runTmpl(t, "version"); code != 0 || !strings.Contains(out, "tmpl dev") {
		t.Errorf("version: exit %d %q", code, out)
	}
}

// /dev/null is a character device but not a terminal; cron, nohup and many
// CI/agent harnesses hand it to processes as stdin.
func TestDevNullStdinNeverPrompts(t *testing.T) {
	devnull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer devnull.Close()
	if isTerminal(devnull) {
		t.Error("isTerminal(/dev/null) = true")
	}
	var out, errb bytes.Buffer
	code := run([]string{fixture(t), filepath.Join(t.TempDir(), "app")}, devnull, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if strings.Contains(out.String(), "Project name") || strings.Contains(out.String(), "[Y/n]") {
		t.Errorf("prompted with /dev/null stdin: %q", out.String())
	}
}
