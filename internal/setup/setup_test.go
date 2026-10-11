package setup

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type recorder struct {
	argv  []string
	chdir string
}

func project(t *testing.T, withSkill bool) string {
	t.Helper()
	dir := t.TempDir()
	if withSkill {
		p := filepath.Join(dir, filepath.FromSlash(SkillPath))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("steps"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func opts(dir, input string, interactive, hasClaude bool, rec *recorder, out *bytes.Buffer) Options {
	return Options{
		Dest: dir, ShownDest: "my-app", Interactive: interactive,
		In:  bufio.NewReader(strings.NewReader(input)),
		Out: out,
		LookPath: func(string) (string, error) {
			if hasClaude {
				return "/bin/claude", nil
			}
			return "", errors.New("not found")
		},
		Chdir: func(d string) error { rec.chdir = d; return nil },
		Exec:  func(_ string, argv []string, _ []string) error { rec.argv = argv; return nil },
	}
}

func TestRunExecsClaudeOnYes(t *testing.T) {
	for _, input := range []string{"\n", "y\n", "YES\n"} {
		dir := project(t, true)
		var rec recorder
		var out bytes.Buffer
		if err := Run(opts(dir, input, true, true, &rec, &out)); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(rec.argv, []string{"claude", Prompt}) || rec.chdir != dir {
			t.Errorf("input %q: argv=%v chdir=%q", input, rec.argv, rec.chdir)
		}
	}
}

func TestRunPrintsHintOnNo(t *testing.T) {
	dir := project(t, true)
	var rec recorder
	var out bytes.Buffer
	if err := Run(opts(dir, "n\n", true, true, &rec, &out)); err != nil {
		t.Fatal(err)
	}
	if rec.argv != nil {
		t.Error("exec'd despite no")
	}
	if !strings.Contains(out.String(), Hint("my-app")) {
		t.Errorf("out = %q", out.String())
	}
}

func TestRunEOFMeansNo(t *testing.T) {
	dir := project(t, true)
	var rec recorder
	var out bytes.Buffer
	if err := Run(opts(dir, "", true, true, &rec, &out)); err != nil {
		t.Fatal(err)
	}
	if rec.argv != nil {
		t.Error("exec'd on EOF")
	}
}

func TestRunNonInteractiveNeverPrompts(t *testing.T) {
	dir := project(t, true)
	var rec recorder
	var out bytes.Buffer
	if err := Run(opts(dir, "y\n", false, true, &rec, &out)); err != nil {
		t.Fatal(err)
	}
	if rec.argv != nil || strings.Contains(out.String(), "[Y/n]") {
		t.Errorf("prompted or exec'd: %q", out.String())
	}
	if !strings.Contains(out.String(), "Next:") {
		t.Errorf("no hint: %q", out.String())
	}
}

func TestRunWithoutClaudePrintsHint(t *testing.T) {
	dir := project(t, true)
	var rec recorder
	var out bytes.Buffer
	if err := Run(opts(dir, "y\n", true, false, &rec, &out)); err != nil {
		t.Fatal(err)
	}
	if rec.argv != nil || !strings.Contains(out.String(), "Next:") {
		t.Errorf("out = %q", out.String())
	}
}

func TestRunWithoutSkillDoesNothing(t *testing.T) {
	dir := project(t, false)
	var rec recorder
	var out bytes.Buffer
	if err := Run(opts(dir, "y\n", true, true, &rec, &out)); err != nil {
		t.Fatal(err)
	}
	if rec.argv != nil || out.Len() != 0 {
		t.Errorf("out = %q", out.String())
	}
}

func TestHint(t *testing.T) {
	if got := Hint("."); strings.HasPrefix(got, "cd ") {
		t.Errorf("Hint(.) = %q", got)
	}
	if got := Hint("my app"); !strings.HasPrefix(got, `cd "my app" && claude `) {
		t.Errorf("Hint(my app) = %q", got)
	}
}
