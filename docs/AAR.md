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

## 2026-05-21 — Book-inspired improvements, transformer caching, special cases, repo reorg

**What went well:**
- Using the book YAMLs as structured inspiration was productive — yielded 4 concrete improvements (io.Writer, signals, benchmarks, distribution) plus 3 backlog items, all grounded in the project's actual needs rather than abstract "we should do X"
- TDD discipline held throughout: every feature started with a failing test. The user caught one skip early (the `-n` implies `-f` fix) which reinforced the habit
- Benchmarks immediately paid off: identified `removeAccents` as the bottleneck (37μs, 21 allocs → 25μs, 6 allocs after caching), and the `specialCases` table refactor was informed by the allocation data
- The special-cases expansion caught a real bug: characters like `ø`, `æ`, `œ` were passing through the pipeline unchanged, producing non-ASCII output despite the tool's promise of filesystem-safe names
- Deliberation-then-decide pattern worked well for config file question — writing the reasoning to the backlog makes the decision durable and reviewable

**What didn't go well:**
- The initial `-n` implies `-f` fix was done without TDD — user had to remind me. Should be automatic by now
- The batch commit (io.Writer + signals + benchmarks + man page + goreleaser) was too large — each feature should have been its own commit for cleaner git history
- Benchmark numbers were noisy on the dev machine (single-core i5, varying wall-clock times across runs) — allocation counts were the reliable metric but that wasn't stated upfront

**What we'll do differently:**
- Never skip TDD, even for "obvious" one-line fixes — the test documents the intent, not just the implementation
- Commit after each logical feature, not in batches — smaller commits are easier to review, revert, and cherry-pick
- When presenting benchmark results, lead with allocation counts (stable) and note wall-clock times as noisy/indicative only

## 2026-05-21 — Postcondition validation, adversarial testing, 80+ special cases

**What went well:**
- The "nuclear power plant" framing drove genuinely rigorous design — postcondition validation, defense-in-depth, diagnostic error messages with codepoints
- TDD discipline was maintained throughout: every feature started with failing tests, including the validation functions, error signatures, and edge cases
- Using a second LLM as adversarial tester was highly effective — it found a panic on empty input and a dotfile edge case that we missed, plus identified 5 unhandled Latin characters
- The three-phase approach (validation → adversarial → expand table) kept each commit focused and reviewable
- Triage of adversarial results was careful: 2 real bugs fixed, 3 wrong expectations corrected with reasoning documented in test comments

**What didn't go well:**
- The `sanitizeFilename("")` panic should have been caught when adding the empty-result validation — we tested empty sanitize output but didn't test empty input to sanitizeFilename
- The `"..hidden"` edge case (dot-only base) was missed until the adversarial LLM found it — `filepath.Ext` behavior on dot-prefixed names deserves more attention
- The adversarial test file initially replaced the entire test suite (wrong package, placeholder bodies) — required recovery from git and manual integration

**What we'll do differently:**
- When adding error handling to a function, systematically test the zero-value/empty input as a first case — it's the most common crash vector
- When accepting LLM-generated test code, always create it as a separate file rather than replacing existing tests — review before integrating
- For `filepath.Ext` edge cases, build a small truth table of inputs (empty, dots-only, leading dots, multiple dots) before coding

## 2026-05-20 — Competitive analysis, refactor renameOne

**What went well:**
- Competitive analysis surfaced that sanitize's niche (zero-config, Latin-only, single binary with full file ops) is genuinely unoccupied — detox is the closest but more complex
- The `renameOne` refactor was clean: -26 net lines, all 210+ tests passed unchanged, proving the test suite is robust against internal restructuring
- Close checklist caught the stale test count in CLAUDE.md (158 vs 210+) and missing `-r` in the POSIX table — the cross-check step earns its keep

**What didn't go well:**
- Nothing significant — this was a short, focused continuation

**What we'll do differently:**
- After adding a feature, immediately grep all docs for the flag set to catch drift (the POSIX table miss was found only during close checklist)
