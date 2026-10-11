// Package setup hands a freshly created project to Claude when the template
// ships setup instructions at .agents/skills/setup/SKILL.md.
package setup

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SkillPath is where templates put plain-English setup steps for Claude.
const SkillPath = ".agents/skills/setup/SKILL.md"

// Prompt is the instruction passed to Claude.
const Prompt = "Follow .agents/skills/setup/SKILL.md to finish setting up this project."

type Options struct {
	Dest        string // absolute project directory
	ShownDest   string // dest as the user typed it, for printed hints
	Interactive bool
	In          *bufio.Reader
	Out         io.Writer
	LookPath    func(string) (string, error)
	Chdir       func(string) error
	Exec        func(argv0 string, argv []string, env []string) error
}

// Run offers to start Claude in the project when the template has a setup
// skill. Interactive with claude installed: ask, and on yes replace this
// process with claude. Otherwise: print the command to run next.
func Run(o Options) error {
	if _, err := os.Stat(filepath.Join(o.Dest, filepath.FromSlash(SkillPath))); err != nil {
		return nil
	}
	if claude, err := o.LookPath("claude"); err == nil && o.Interactive {
		fmt.Fprint(o.Out, "Run Claude to finish setup? [Y/n] ")
		line, readErr := o.In.ReadString('\n')
		ans := strings.ToLower(strings.TrimSpace(line))
		eof := readErr != nil && line == ""
		if !eof && (ans == "" || ans == "y" || ans == "yes") {
			if err := o.Chdir(o.Dest); err != nil {
				return err
			}
			return o.Exec(claude, []string{"claude", Prompt}, os.Environ())
		}
	}
	fmt.Fprintln(o.Out, "Next: "+Hint(o.ShownDest))
	return nil
}

// Hint is the shell command that starts Claude on the project's setup skill.
func Hint(shownDest string) string {
	cmd := fmt.Sprintf("claude %q", Prompt)
	if shownDest == "" || shownDest == "." {
		return cmd
	}
	if strings.ContainsAny(shownDest, " \t'\"$") {
		shownDest = fmt.Sprintf("%q", shownDest)
	}
	return fmt.Sprintf("cd %s && %s", shownDest, cmd)
}
