# tmpl — design

Date: 2026-10-09
Status: draft, awaiting review

## Purpose

Starting a new app from a blank slate is annoying. `tmpl` copies a project
template into a directory with one command, on any machine:

```sh
tmpl pierre .
```

It is built for the author first (their own templates, one command), but
accepts any local path or GitHub source so other people can use it with their
own templates without forking it.

### Success criteria

- `brew install wsoule/tap/tmpl` then `tmpl pierre my-app` produces a working
  project directory on a fresh macOS or Linux machine.
- Someone else can run `tmpl someone/their-templates/web my-app` without
  configuring anything.
- After copying, the project is named after the target (not "template"), and
  Claude can finish open-ended setup from instructions shipped in the template.

### Non-goals (v1)

- Updating existing projects when a template changes (copier-style sync).
- A templating language (conditionals, loops, prompts beyond the project name).
- An npm package, Windows support, or a homebrew-core submission.
- Fetching from hosts other than GitHub.

## Repository layout

One repo, `github.com/wsoule/tmpl`, holds the CLI and the author's templates:

```
tmpl/
├── cmd/tmpl/main.go          # flag parsing, wiring, exit codes
├── internal/
│   ├── source/               # parse a source string → Source
│   ├── fetch/                # Source → files on disk (local dir or GitHub tarball)
│   ├── copy/                 # write files into dest (skip rules, no-overwrite)
│   ├── rename/               # {{name}} substitution
│   └── setup/                # Claude handoff
├── templates/
│   ├── pierre/
│   └── bun-typescript-monorepo/
├── .goreleaser.yaml
└── .github/workflows/{ci,release}.yml
```

`templates/` contains no `.go` files, so the Go toolchain ignores it.

### Migrating the existing templates

- `templates/pierre/` ← current contents of `wsoule/pierre-template` (no `.git`,
  no `node_modules`).
- `templates/bun-typescript-monorepo/` ← current contents of
  `wsoule/bun-ts-mono-template`.
- The old repos are left alone (not archived or deleted) in v1.
- Trade-off accepted: `pierre` no longer lives in a git fork of upstream
  `pierrecomputer/pierre`, so upstream changes are pulled in by hand.

## CLI

```
tmpl <source> [dest]      copy a template into dest (default ".")
tmpl list                 list templates in the default repo
tmpl version
```

Flags for the copy command:

| Flag | Meaning |
|---|---|
| `--name <n>` | project name for `{{name}}`; default: base name of dest's absolute path |
| `--force` | overwrite existing files in dest |
| `--no-setup` | skip the Claude handoff |
| `-y`, `--yes` | accept defaults, never prompt (also implied when stdin is not a TTY) |

Exit codes: `0` success, `1` usage error, `2` source not found, `3` other
failure (network, I/O).

## Source resolution (`internal/source`)

A source string is resolved in this order:

1. **Local path** — starts with `/`, `./`, `../`, or `~`, or is an existing
   directory relative to the current working directory. Used as-is.
2. **GitHub path** — contains a `/`: `owner/repo[/subdir...][@ref]`.
   `ref` defaults to the repo's default branch.
3. **Bare name** — `pierre` → `<default repo>/templates/pierre`, where the
   default repo is `wsoule/tmpl` unless the `TMPL_REPO` environment variable
   is set (e.g. `TMPL_REPO=alice/templates`). This is how other people make
   bare names point at their own templates.

## Fetching (`internal/fetch`)

- **Local:** walk the directory directly.
- **GitHub:** stream `https://codeload.github.com/<owner>/<repo>/tar.gz/<ref>`
  (ref `HEAD` when unset), gunzip, read the tar, and keep only entries under
  `<top-level dir>/<subdir>/`, stripping that prefix. The whole repo is
  downloaded; templates are small, and this avoids the GitHub API rate limit.
- **Private repos:** if `GITHUB_TOKEN` is set, or `gh auth token` succeeds,
  send it as a bearer token. Otherwise fetch anonymously.
- **Not found:** an HTTP 404, or a subdir with no entries, is exit code 2 with
  the message `template "<x>" not found`, followed by `tmpl list` output when
  the source was a bare name.
- **Safety:** reject tar entries whose cleaned path is absolute or escapes the
  destination (`..`). Keep symlinks only if their target stays inside dest;
  skip others with a warning. Skip device and fifo entries.

## Copying (`internal/copy`)

- Always skip `.git/`, `node_modules/`, `.DS_Store`, and `tmpl.json` (the
  template's own manifest).
- Create `dest` if missing.
- Without `--force`, existing files are never overwritten. They are listed at
  the end ("kept existing: …"). With `--force`, they are replaced.
- Preserve file modes (the executable bit matters for scripts).

## Renaming (`internal/rename`)

- The project name comes from `--name`, else a prompt (default: base name of
  dest) when interactive, else the default with no prompt.
- In every copied **text** file, replace the literal token `{{name}}` with the
  project name. A file is text if its first 8 KB contains no NUL byte.
  Replacement only touches files written in this run, never pre-existing
  files.
- Template authors add `{{name}}` where it belongs (e.g. `"name": "{{name}}"`
  in `package.json`, the moon project id). Part of migration is adding these
  tokens to `pierre` and `bun-typescript-monorepo`.
- Validation: the name must match `^[a-z0-9][a-z0-9._-]*$` (npm-safe). An
  invalid name is a usage error that suggests a fixed-up version.

## Claude handoff (`internal/setup`)

- Templates may ship `.agents/skills/setup/SKILL.md`: plain-English setup
  steps for that template (what to rename beyond `{{name}}`, which example
  files to delete, how to install and run). It is copied like any other file,
  matching the `.agents/skills/` layout the templates already use.
- After copying, if that file exists in dest and `--no-setup` is not set:
  - **Interactive and `claude` on PATH:** prompt
    `Run Claude to finish setup? [Y/n]`. On yes, replace the tmpl process with
    `claude "Follow .agents/skills/setup/SKILL.md to finish setting up this project."`
    in dest (`syscall.Exec`).
  - **Otherwise:** print the next step:
    `cd <dest> && claude "Follow .agents/skills/setup/SKILL.md …"`.
- Templates without a setup skill end after copying, with a one-line summary.

## Template manifest (`tmpl.json`, optional)

```json
{ "description": "Bun + moon TypeScript monorepo (Pierre conventions)" }
```

Used only by `tmpl list`. It is never copied into the project.

## `tmpl list`

Fetches the default repo's tarball (same code path as a fetch) and prints each
`templates/*` directory with its `tmpl.json` description. No API calls.

## Distribution

- GoReleaser, modeled on `wsoule/stack`: on a `v*` tag, build darwin/linux ×
  amd64/arm64 with `CGO_ENABLED=0`, create the GitHub release, and write the
  formula to `wsoule/homebrew-tap`.
- The tap push uses a `HOMEBREW_TAP_TOKEN` repo secret (a fine-grained PAT
  with contents:write on `homebrew-tap`). The default `GITHUB_TOKEN` cannot
  push across repos. Use whichever Homebrew publisher the current GoReleaser
  v2 recommends (`brews` is being deprecated in favor of newer config); verify
  at implementation time.
- Install: `brew install wsoule/tap/tmpl`. Also `go install
  github.com/wsoule/tmpl/cmd/tmpl@latest`.

## Testing

- **Unit:** source parsing table tests (every form above, including
  `TMPL_REPO`); tar extraction against in-memory fixture tarballs (subdir
  filtering, prefix stripping, `..` and absolute path rejection, symlink
  escape); copy skip rules and no-overwrite behavior; binary detection and
  `{{name}}` replacement; name validation.
- **Integration:** run the built binary against a local fixture template in a
  temp dir with `--yes --no-setup` and assert the output tree. The GitHub
  fetcher is tested with `httptest.Server` serving a fixture tarball (the base
  URL is injectable).
- **CI:** `go vet`, `go test -race ./...` on push/PR.
- **Manual check before the first tag:** `tmpl pierre /tmp/x` against the real
  repo, then `pnpm install` and the template's own verify tasks in `/tmp/x`.

## External actions (need explicit go-ahead at the time)

1. Create the public repo `wsoule/tmpl` and push.
2. Add the `HOMEBREW_TAP_TOKEN` secret (the user creates the PAT).
3. Push the first `v0.1.0` tag, which publishes the release and the tap
   formula.
