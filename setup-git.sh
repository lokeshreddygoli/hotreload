#!/usr/bin/env bash
# setup-git.sh — run once to initialize the repo with a clean commit history
# Usage: bash setup-git.sh

set -e

echo "==> Configuring git identity..."
git config user.name "Lokesh Reddy"
git config user.email "lokeshreddygoli@gmail.com"   # update if different

echo "==> Initializing repo..."
git init
git checkout -b main

echo "==> Initial commit — project scaffold..."
git add go.mod .gitignore
git commit -m "init: project scaffold with go.mod"

echo "==> Add CLI entrypoint..."
git add main.go cmd/
git commit -m "feat(cmd): add hotreload CLI with --root, --build, --exec flags"

echo "==> Add file filter..."
git add internal/filter/
git commit -m "feat(filter): ignore .git, vendor, editor swap files and build artifacts"

echo "==> Add debouncer..."
git add internal/debounce/
git commit -m "feat(debounce): 300ms quiet-window debouncer to coalesce rapid saves"

echo "==> Add process manager..."
git add internal/process/
git commit -m "feat(process): platform-aware process manager with full tree kill

Unix: Setpgid + SIGTERM -> SIGKILL escalation
Windows: Job Objects with taskkill /F /T fallback"

echo "==> Add watcher..."
git add internal/watcher/
git commit -m "feat(watcher): fsnotify wrapper with dynamic dir watching and inotify cap"

echo "==> Add engine..."
git add internal/engine/
git commit -m "feat(engine): main rebuild loop with build cancellation and crash backoff

- buffered trigger channel coalesces in-flight events
- per-cycle context cancels in-progress builds on new trigger
- generation counter prevents stale cycle from clobbering crash count
- 5s backoff after 3 consecutive fast crashes"

echo "==> Add test server..."
git add testserver/
git commit -m "chore(testserver): add demo HTTP server for hot reload walkthrough"

echo "==> Add Makefile and docs..."
git add Makefile README.md LICENSE docs/ .github/
git commit -m "chore: add Makefile, README, LICENSE, CI workflow, and GitHub Pages site"

echo ""
echo "Done. Push with:"
echo "  git remote add origin https://github.com/lokeshreddygoli/hotreload.git"
echo "  git push -u origin main"