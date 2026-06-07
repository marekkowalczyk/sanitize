# Next Session

- **Tag v1.0.0**: `git tag v1.0.0 && git push origin v1.0.0` — repo is ready, goreleaser will build cross-platform binaries. Create CHANGELOG.md at this point.
- **Migrate `/usr/local/bin/san`**: Replace legacy `san.sh` with symlink to Go binary. Test first since `san` is used daily.
- **Pre-scan collision detection** (backlog, high priority): Build full old→new mapping before renaming to prevent partial-rename half-states with `-r`.
