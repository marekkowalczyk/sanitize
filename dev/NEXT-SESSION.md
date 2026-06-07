# Next Session

- **Review v3.0.0 transliteration choices** — deliberate on design decisions from 2026-06-07 session. See dev/BACKLOG.md for full list of open questions. May result in reverting some choices.
- **Check goreleaser** — verify v2.0.0 and v3.0.0 releases built correctly at GitHub Actions.
- **San migration cleanup** — symlink set up 2026-05-21, backup at `/usr/local/bin/san.sh.bak`. Remove backup if confident.
- **Pre-scan collision detection** — highest priority safety feature (dev/BACKLOG.md). Should be done before recommending `-r` for production use on large trees.
