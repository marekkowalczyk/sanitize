# Next Session

- **Tag v1.0.0** — create CHANGELOG.md summarizing all pre-release work, then `git tag v1.0.0 && git push origin v1.0.0`. Triggers goreleaser to build cross-platform binaries and publish a GitHub Release.
- **Pre-scan collision detection** — highest priority safety feature (docs/BACKLOG.md). Build old→new mapping, detect duplicate targets, abort before any renames. Should be done before recommending `-r` for production use on large trees. Consider doing this before v1.0.0 if scope allows.
- **Verify san migration** — old san.sh replaced with symlink on 2026-05-21. Use `san` in daily workflow for a few days to confirm no regressions. Backup at `/usr/local/bin/san.sh.bak` can be removed once confident.
