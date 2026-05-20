# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go CLI tool that sanitizes/normalizes strings for safe use as filenames. It lowercases, strips diacritics, replaces non-alphanumeric characters with hyphens, deduplicates hyphens, and trims ends.

## Build and Run

```bash
go build sanitize.go
./sanitize "input text here"
```

Dependencies are managed via Go modules (`go.mod`). The main external dependency is `golang.org/x/text` for Unicode normalization and rune manipulation.

## Architecture

Single-file Go program (`sanitize.go`) with a transformation pipeline composed via function nesting:

```
input → removeIllFormed → toLower → removeAccents → replaceNonAlphaNum → dedupHyp → trimEnds → output
```

Key design decisions:
- Diacritics are stripped using Unicode NFD decomposition + removal of `unicode.Mn` (Mark, Nonspacing) category runes
- Polish `ł`/`Ł` are special-cased with direct string replacement because they are standalone characters, not combining sequences
- `replaceNonAlphaNum` uses `unicode.Latin` (not `unicode.Letter`) to restrict output to Latin script characters only
- Multiple CLI arguments are joined with `-` before processing

## Supporting Scripts

- `san.sh` — Bash wrapper that splits filenames from extensions, sanitizes both parts separately, and renames files (won't overwrite)
- `DEVONthink-Sanitize-Filenames.applescript` — AppleScript for sanitizing DEVONthink record names (saves originals in Finder Comment field)
