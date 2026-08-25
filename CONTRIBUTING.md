# Contributing

Thanks for helping improve `differ`.

## Scope

Keep the project focused: one fast, readable, side-by-side Git diff view. New
features should improve reviewing a diff without adding general Git management
screens.

## Development

```bash
git clone https://github.com/furqanalishah/differ.git
cd differ
go test ./...
go vet ./...
go build -o differ .
```

Format Go files with `gofmt` before committing.

## Pull requests

- Explain the review problem the change solves.
- Add or update tests for behavior changes.
- Update `CHANGELOG.md` for user-visible changes.
- Keep unrelated changes separate.
- Confirm `go test ./...`, `go vet ./...`, and `go build .` pass.
