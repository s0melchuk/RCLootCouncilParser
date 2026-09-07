# Contributing

This is a small personal/guild-utility project, but issues and PRs are
welcome.

## Setup

```bash
go build -o rclootparser .
./rclootparser init
```

No external dependencies — standard library only. Keep it that way unless
there's a strong reason to add one (this is a small CLI meant to build and
run anywhere without a network fetch).

## Before opening a PR

- Run `gofmt -l .` (should be empty) and `go vet ./...` — both are enforced
  in CI.
- Run `go test ./...` and add tests for parsing changes — `internal/luatable`,
  `internal/savedvars`, and `internal/chatlog` are the highest-risk,
  hand-written bits (Lua table literals, regex extraction) and deserve
  coverage for any format variant you encounter.
- Don't commit real secrets — `config.json` (with a real API key) or
  `state.json` belong nowhere in the repo. See [SECURITY.md](SECURITY.md).
- If you touch the chat-log or SavedVariables parsing, include a redacted
  real sample (or a synthetic one shaped like real RCLootCouncil output) in
  the PR description or as a test fixture — these formats come from the
  addon's own Lua source, not a spec, so a real example is the only ground
  truth.

## Reporting bugs / requesting features

Use the issue templates — they'll prompt for what's needed to act on it.
