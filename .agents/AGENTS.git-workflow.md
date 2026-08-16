# Git Repository Workflow

- Canonical repository: `https://github.com/shishtpal/browser-server.git`.
- Start work from an up-to-date `main`: fetch the remote, then create a focused branch such as `feat/short-description`, `fix/short-description`, or `docs/short-description`.
- Before editing, inspect `git status` and preserve unrelated user or agent changes. Never discard, overwrite, or stage files outside the requested work.
- Keep commits focused and use the repository's Conventional Commit-style subjects: `feat(scope): ...`, `fix(scope): ...`, `docs(scope): ...`, `refactor(scope): ...`, or `chore(scope): ...`.
- Run checks appropriate to the changed area before committing. At minimum, use `go test ./...` and `go vet ./...` for backend changes, the package build for frontend changes, and `type-check` plus the package build for extension changes.
- Review both `git diff` and `git diff --cached` so commits contain only intended changes.
- Never commit generated output, dependencies, runtime data, or secrets: `bin/`, `dist/`, `node_modules/`, `.data/`, `.bs-token`, `.env`, and local logs.
- Push feature branches and open pull requests against `main`; do not force-push or rewrite shared branch history.
- For fork-based work, use the fork as `origin` and add this repository as `upstream`: `git remote add upstream https://github.com/shishtpal/browser-server.git`.
