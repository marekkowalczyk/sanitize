# Next Session

- **Pre-scan collision detection** — highest priority safety feature before `-r` is used on real trees. Build old->new mapping, detect duplicates, abort before any renames. See BACKLOG.md (in this directory).
- **Tag v1.0.0** — `git tag v1.0.0 && git push origin v1.0.0`. Then `go install github.com/marekkowalczyk/sanitize@v1.0.0` works for anyone.
- **Migrate /usr/local/bin/san** — replace the old san.sh with a symlink to the Go binary. Steps in README under "Migrating from san.sh". Test carefully first.
