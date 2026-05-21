# Next Session

- **Tag v1.0.0** — `git tag v1.0.0 && git push origin v1.0.0`. Triggers goreleaser to build cross-platform binaries and publish a GitHub Release. Then `go install github.com/marekkowalczyk/sanitize@v1.0.0` works for anyone.
- **Migrate /usr/local/bin/san** — replace the old san.sh with a symlink to the Go binary. Steps in README under "Migrating from san.sh". Test carefully first since san is used several times daily.
- **Pre-scan collision detection** — highest priority safety feature (docs/BACKLOG.md). Build old→new mapping, detect duplicate targets, abort before any renames. Should be done before recommending `-r` for production use on large trees.
