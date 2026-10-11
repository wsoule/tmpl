# tmpl

Start new projects from templates with one command.

```sh
brew install wsoule/tap/tmpl
tmpl pierre my-app
```

## Usage

```
tmpl <source> [dest]   copy a template into dest (default ".")
tmpl list              list templates in the default repo
tmpl version
```

`<source>` can be:

| Source | Example |
|---|---|
| A template name | `tmpl pierre` → `templates/pierre` in `wsoule/tmpl` |
| A GitHub repo or folder | `tmpl alice/templates/web@v2` |
| A local directory | `tmpl ./my-template`, `tmpl ~/templates/web` |

Flags: `-n/--name` sets the project name, `-f/--force` overwrites existing
files, `--no-setup` skips the Claude handoff, and `-y/--yes` never prompts.

Existing files are never overwritten unless you pass `--force`, so it's safe
to run in a folder that already has things in it.

## Use your own templates

Make a repo with a `templates/` folder (one subfolder per template), then:

```sh
export TMPL_REPO=you/your-templates
tmpl list
tmpl web my-app
```

Or skip the env var and use the full path: `tmpl you/your-templates/templates/web`.

### Template features (all optional)

- **`{{name}}`** anywhere in a text file is replaced with the project name.
  The default is the folder name, lowercased and made package-safe.
- **`tmpl.json`**: `{ "description": "…" }`, shown by `tmpl list`. It is never
  copied into projects.
- **`.agents/skills/setup/SKILL.md`**: plain-English setup steps. After
  copying, `tmpl` offers to start [Claude Code](https://claude.com/claude-code)
  in the project to follow them.

`.git`, `node_modules` and `.DS_Store` are never copied. Private repos work if
`GITHUB_TOKEN` is set or you're logged in with `gh`.

## Install

```sh
brew install wsoule/tap/tmpl
# or
go install github.com/wsoule/tmpl/cmd/tmpl@latest
```
