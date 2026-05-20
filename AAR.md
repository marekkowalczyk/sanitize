# After Action Review

Continuous improvement log. Each session ends with a brief review: what went well, what didn't, what to change. This is the POOGI (Process Of Ongoing Improvement) record for this project.

## 2026-05-20 — Add recursive rename mode, collision risk docs, backlog

**What went well:**
- TDD continued to work smoothly — 6 new tests written first, all caught the initial "unknown flag" failure, then all passed after implementation
- Depth-first traversal for `-r` was the right design call; handled naturally by reversing the `filepath.Walk` order
- User's collision concern was a valuable catch — the no-clobber check does prevent data loss, but the partial-rename problem is real and worth documenting

**What didn't go well:**
- Context ran out mid-session (carried over from prior conversation), requiring a cold restart from a summary — some ramp-up cost re-reading files
- Initial test for `-r` without `-f` assumed it should error, then immediately changed to "implies file mode" — could have thought through the UX before writing the test

**What we'll do differently:**
- For new flags, decide the interaction with existing flags *before* writing tests (sketch the flag matrix)
- Pre-scan collision detection should be implemented before recommending `-r` for production use on large trees — prioritize this in the next session
