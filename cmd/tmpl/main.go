// Command tmpl copies a project template into a directory.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/wsoule/tmpl/internal/catalog"
	"github.com/wsoule/tmpl/internal/fetch"
	"github.com/wsoule/tmpl/internal/rename"
	"github.com/wsoule/tmpl/internal/setup"
	"github.com/wsoule/tmpl/internal/source"
	"github.com/wsoule/tmpl/internal/write"
)

// Version is set at release time with -ldflags "-X main.Version=...".
var Version = "dev"

const (
	exitOK       = 0
	exitUsage    = 1
	exitNotFound = 2
	exitFail     = 3
)

const usage = `usage:
  tmpl <source> [dest]   copy a template into dest (default ".")
  tmpl list              list templates in the default repo
  tmpl version

source:
  pierre                      a template in $TMPL_REPO (default wsoule/tmpl)
  owner/repo[/path][@ref]     any GitHub repo or folder in one
  ./path  ~/path  /path       a local directory

flags:
  -n, --name <name>   project name for {{name}} (default: dest folder name)
  -f, --force         overwrite existing files
      --no-setup      don't offer to run Claude afterwards
  -y, --yes           never prompt
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is main without os.Exit, so tests can call it in-process.
func run(argv []string, stdin *os.File, stdout, stderr io.Writer) int {
	a, err := parseArgs(argv)
	if err != nil {
		fmt.Fprintf(stderr, "tmpl: %v\n\n%s", err, usage)
		return exitUsage
	}
	switch a.cmd {
	case "help":
		fmt.Fprint(stdout, usage)
		return exitOK
	case "version":
		fmt.Fprintln(stdout, "tmpl "+Version)
		return exitOK
	case "list":
		return runList(stdout, stderr)
	}
	return runCopy(a, stdin, stdout, stderr)
}

func runCopy(a args, stdin *os.File, stdout, stderr io.Writer) int {
	ctx := context.Background()
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "tmpl: %v\n", err)
		return exitFail
	}
	home, _ := os.UserHomeDir()

	if a.name != "" {
		if err := rename.Validate(a.name); err != nil {
			fmt.Fprintf(stderr, "tmpl: %v\n", err)
			return exitUsage
		}
	}

	src, err := source.Parse(a.source, source.Options{Cwd: cwd, Home: home, DefaultRepo: source.RepoFromEnv()})
	if err != nil {
		fmt.Fprintf(stderr, "tmpl: %v\n", err)
		return exitUsage
	}

	gh := fetch.NewGitHub("")
	if src.Kind == source.GitHub {
		gh.Token = fetch.TokenFromEnv()
	}
	res, err := fetch.Fetch(ctx, src, gh)
	if errors.Is(err, fetch.ErrNotFound) {
		fmt.Fprintf(stderr, "tmpl: template %q not found\n", a.source)
		if hint := localHint(a.source, cwd); hint != "" {
			fmt.Fprintf(stderr, "tmpl: %s\n", hint)
		}
		if src.Bare {
			fmt.Fprintln(stderr, "\navailable templates:")
			runList(stderr, io.Discard)
		}
		return exitNotFound
	}
	if err != nil {
		fmt.Fprintf(stderr, "tmpl: %v\n", err)
		return exitFail
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(stderr, "tmpl: warning: %s\n", w)
	}

	dest := a.dest
	if !filepath.IsAbs(dest) {
		dest = filepath.Join(cwd, dest)
	}
	interactive := !a.yes && isTerminal(stdin)
	in := bufio.NewReader(stdin)

	if rename.Uses(res.Files) {
		name, err := projectName(a.name, dest, interactive, in, stdout)
		if err != nil {
			fmt.Fprintf(stderr, "tmpl: %v\n", err)
			return exitUsage
		}
		rename.Apply(res.Files, name)
	}

	rep, err := write.Files(dest, res.Files, a.force)
	if err != nil {
		fmt.Fprintf(stderr, "tmpl: %v\n", err)
		return exitFail
	}
	fmt.Fprintf(stdout, "Created %s from %s (%s)\n", a.dest, a.source, files(len(rep.Written)))
	if len(rep.Kept) > 0 {
		fmt.Fprintf(stdout, "Kept %s already in %s (use --force to overwrite):\n", files(len(rep.Kept)), a.dest)
		for _, p := range rep.Kept {
			fmt.Fprintf(stdout, "  %s\n", p)
		}
	}

	if !a.noSetup {
		err := setup.Run(setup.Options{
			Dest: dest, ShownDest: a.dest, Interactive: interactive,
			In: in, Out: stdout,
			LookPath: exec.LookPath, Chdir: os.Chdir, Exec: syscall.Exec,
		})
		if err != nil {
			fmt.Fprintf(stderr, "tmpl: %v\n", err)
			return exitFail
		}
	}
	return exitOK
}

// files formats a file count for messages: "1 file", "3 files".
func files(n int) string {
	if n == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", n)
}

// localHint helps when a remote template isn't found but a folder of that name
// is in the current directory. Local folders need a "./" prefix (so they never
// shadow template names), and someone typing "tmpl web" may have meant ./web.
func localHint(raw, cwd string) string {
	if strings.HasPrefix(raw, ".") || strings.HasPrefix(raw, "~") || filepath.IsAbs(raw) {
		return ""
	}
	if fi, err := os.Stat(filepath.Join(cwd, raw)); err != nil || !fi.IsDir() {
		return ""
	}
	return fmt.Sprintf("did you mean the local folder %q?", "./"+raw)
}

// projectName decides what replaces {{name}}: the --name flag, else the
// user's answer to a prompt, else the dest folder name normalized to a valid name.
func projectName(flag, dest string, interactive bool, in *bufio.Reader, out io.Writer) (string, error) {
	if flag != "" {
		return flag, nil // validated before fetching
	}
	def := rename.Suggest(filepath.Base(dest))
	if !interactive {
		return def, nil
	}
	for {
		fmt.Fprintf(out, "Project name (%s): ", def)
		line, readErr := in.ReadString('\n')
		name := strings.TrimSpace(line)
		if name == "" {
			return def, nil
		}
		if err := rename.Validate(name); err != nil {
			if readErr != nil {
				return "", err
			}
			fmt.Fprintln(out, err)
			continue
		}
		return name, nil
	}
}

func runList(stdout, stderr io.Writer) int {
	src, err := source.Templates(source.RepoFromEnv())
	if err != nil {
		fmt.Fprintf(stderr, "tmpl: %v\n", err)
		return exitUsage
	}
	res, err := fetch.NewGitHub(fetch.TokenFromEnv()).Fetch(context.Background(), src.Owner, src.Repo, src.Ref, src.Subdir)
	if err != nil {
		fmt.Fprintf(stderr, "tmpl: %v\n", err)
		if errors.Is(err, fetch.ErrNotFound) {
			return exitNotFound
		}
		return exitFail
	}
	tw := tabwriter.NewWriter(stdout, 0, 0, 3, ' ', 0)
	for _, e := range catalog.List(res.Files) {
		fmt.Fprintf(tw, "  %s\t%s\n", e.Name, e.Description)
	}
	tw.Flush()
	return exitOK
}

// isTerminal reports whether f is an interactive terminal: a character
// device other than /dev/null, which cron, nohup and CI harnesses often use
// as stdin.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	null, err := os.Stat(os.DevNull)
	return err != nil || !os.SameFile(fi, null)
}
