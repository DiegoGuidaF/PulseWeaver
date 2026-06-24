# Contributing to PulseWeaver

Thanks for your interest in contributing! This document covers how to get set up, the conventions used in the project, and how to submit changes.

## Getting started

1. **Fork** the repository and clone your fork
2. Install hooks (required — validates commit messages):
   ```sh
   make install-hooks
   ```
3. Install dependencies and start developing:
   ```sh
   cp .env.example .env   # fill in values
   make back-dev          # Go backend with hot reload
   make front-dev         # React frontend with Vite
   ```

See the [README](README.md) for full setup instructions.

## Commit messages

This project uses [Conventional Commits](https://www.conventionalcommits.org/). The `commit-msg` hook (installed via `make install-hooks`) enforces this automatically.

**Format:** `type[(scope)]: description`

| Type | When to use |
|------|-------------|
| `feat` | A new feature |
| `fix` | A bug fix |
| `perf` | Performance improvement |
| `refactor` | Code change that is neither a feature nor a fix |
| `docs` | Documentation only |
| `test` | Adding or correcting tests |
| `chore` | Build, tooling, or maintenance |
| `ci` | CI/CD changes |

**Scopes** are optional and indicate the area of the codebase: `ui`, `backend`, `ai`, `ci`, `db`. Omit scope for cross-cutting changes.

```sh
feat(ui): add device detail tab for address history
fix(backend): handle nil pointer in forward-auth handler
feat!: redesign policy engine API (breaking change)
feat: update heartbeat endpoint and client config together
```

## Code style

- **Backend (Go):** explicit, boring Go — no magic, early returns, small focused functions. Keep the layering intact (handler → service → repository → DB); domain constructors enforce invariants, and OpenAPI types stay at the transport edge. See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for how the layers fit together.
- **Frontend (TypeScript/React):** ESLint and TypeScript strict mode. Run `make front-lint`.
- All code must pass `make check` (lint + type-check + tests) before submitting a PR.

## Submitting a pull request

1. Create a branch from `main`: `git checkout -b feat/my-feature`
2. Make your changes and commit following the convention above
3. Run `make check` — all checks must pass
4. Push your branch and open a PR against `main`
5. Fill in the PR template

**Scope of PRs:** Keep PRs focused on a single concern. Large, mixed-purpose PRs are harder to review and slower to merge.

## Running tests

```sh
make back-test    # backend tests
make front-test   # frontend tests
make check        # lint + typecheck + all tests
```

## Questions

For questions and discussion, open a [GitHub Discussion](https://github.com/DiegoGuidaF/PulseWeaver/discussions) rather than an issue.
