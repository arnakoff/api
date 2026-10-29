<!-- Title: Conventional Commits, e.g. `feat(signal): flag domains registered less than 30 days ago`. It becomes the squash commit and drives the release. -->

## Changes

<!-- One bullet per behaviour. Quote any user-visible text (signal messages, error details). -->

-

## Verification

<!-- Commands run and their result, benchmark figures (`benchstat` against main) when the hot path changed, eval figures when a rule or source changed, coverage of touched `domain` and `app` packages, what is not verified. -->

- [ ] `go build ./...`
- [ ] `go vet ./...`
- [ ] `golangci-lint run`
- [ ] `go test -race -shuffle=on ./...`
- [ ] `go tool govulncheck ./... tool`
- [ ] External contributors: I have signed the [contributor licence agreement](https://cla-assistant.io/arnakoff/api)

Part of #

Closes #
