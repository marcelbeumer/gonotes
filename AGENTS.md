# AGENTS.md

## Project Overview

A zettelkasten-like note-taking system backed by plain markdown files and symlinks.

## How the program works and it's intent

Read the README.md for details.

## Build & Test

There is no Makefile. Just use standard go commands.

- Run: `go run ./cmd/gonotes`
- Build: `go build ./cmd/gonotes`
- Test: `go test -v -p 1 ./...`
- Vet: `go vet ./...`
- Format: use gofumpt.

## Code Standards

- [Google Go Style Guide](https://google.github.io/styleguide/go/) (itself a superset of Effective Go)
- General spirit of idiomatic Go: keeping it simple and concise.

## Testing Requirements

- All features should be covered by tests.
