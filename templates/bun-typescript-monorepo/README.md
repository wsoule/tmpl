# {{name}}

This repository is a starter template for new local projects. It keeps the
workspace tooling, TypeScript defaults, linting, formatting, CI, hooks, and
agent instructions, without any product-specific source code.

## Start

```bash
bun install
bun run format
bun run lint
bun run tsc
bun run test
```

## Workspace Commands

Use `bun ws` to run package scripts from the repository root:

```bash
bun ws template build
bun ws template test
bun ws "packages/*" tsc --sequential
```

New libraries belong under `packages/`. New apps belong under `apps/`.
