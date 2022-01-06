# notes-in-go

Port of personal notes system from TS to Go, for learning purposes.  
I made [some notes](./about_go.md) on my first experiences writing Go.

# Install

```
go install github.com/marcelbeumer/notes-in-go/notes@latest
```

or

```
go install github.com/marcelbeumer/notes-in-go/notes@<revision>
```

`@latest` does not seem to upgrade well when running it to upgrade, will read into publishing modules later and publish versions properly.

# Run from source

- run `go mod tidy` inside `notes` folder
- `./scripts/examples` to build and run against `examples` folder
