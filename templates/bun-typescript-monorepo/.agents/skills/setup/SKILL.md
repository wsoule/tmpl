---
name: setup
description: One-time setup for a project just created from the bun-typescript-monorepo template with tmpl. Use only when asked to finish setting up this project; delete this skill when done.
---

# Finish setting up this project

This project was just created from the `bun-typescript-monorepo` template.
The root `package.json` `name` is usually already set. If the folder already had
files, tmpl kept them instead of the template's copies: never overwrite or
replace a file the user already had without asking. Do the rest, confirming with
the user where noted.

1. **Learn the project.** Read `AGENTS.md`, `README.md`, and the root
   `package.json` `name` (that is the project name). Ask the user in one or two
   sentences what they're building, unless they've already said.
2. **Rename the workspace scope.** Packages use the `@workspace/` scope.
   Replace `@workspace/` with `@<project-name>/` in every `package.json`,
   `tsconfig*.json`, Markdown doc, and source import. Don't edit `bun.lock`; it is
   regenerated in step 5.
3. **Decide on the example package.** `packages/template` is a placeholder.
   Ask whether to rename it into the user's first real package (and to what) or
   delete it.
4. **Update docs.** Ask before replacing the README body; if it is still the
   template's README, replace it with a short description of the user's
   project and how to run it. In `AGENTS.md`, change the
   "Bun TypeScript Monorepo Template" title to the project name.
5. **Install and verify.**
   ```bash
   bun install
   bun run format
   bun run lint
   bun run tsc
   bun run test
   ```
   Fix anything the renames broke.
6. **Git.** If this isn't a git repo yet, ask before running `git init` and
   making an initial commit. Husky's `prepare` hook needs a git repo.
7. **Clean up.** Delete `.agents/skills/setup/`, then tell the user what you
   changed and what to run next.
