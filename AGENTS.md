# AGENTS Instructions for ChatWire

These instructions apply to all files in this repository.

## Required checks
- Run `go vet ./...` for static analysis.
- Run unit tests with `go test ./...`.

## Formatting
- Ensure all `.go` files are formatted with `gofmt -w`.

## Documentation
- Update `README.md` when you add new flags or change usage examples. The readme should be concise but friendly 'golang style'.
- Update ATTRIBUTION.md if you add or remove imports (check go.mod)

## Commits and PRs
- Use concise commit messages (first line under 72 characters).
- Reference relevant files or lines in your pull request summary.
