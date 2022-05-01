# gonotes

Personal notes system written in Go. Port of older tool written in TypeScript.

# Install

- `go install github.com/marcelbeumer/gonotes@v1.0.0`

# Usage

- `mkdir notes`
- `cd notes`
- `touch .is_gonotes_root`
- `gonotes new --title "Example note" --tag topic/subtopic --tag special`
- `gonotes new --tag bookmark/go --href="https://go.dev" --scrape`
- `gonotes rename-tag --from bookmark/go --to bookmark/dev`
- `gonotes last` to get last created note
- `gonotes sync` when manually adding notes or changing metadata
