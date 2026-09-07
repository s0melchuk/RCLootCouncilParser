# RCLootCouncilParser

[![CI](https://github.com/s0melchuk/RCLootCouncilParser/actions/workflows/ci.yml/badge.svg)](https://github.com/s0melchuk/RCLootCouncilParser/actions/workflows/ci.yml)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)

A native scanner that watches [RCLootCouncil](https://www.curseforge.com/wow/addons/rclootcouncil)'s
own data and syncs loot awards to [RCLootCouncilApi](https://github.com/s0melchuk/RCLootCouncilApi)
(`POST /api/loot`) — no manual data entry, no forced `/reload`.

## Why two data sources

RCLootCouncil already keeps a structured award history in its own
SavedVariables file, but **that file is only written to disk on logout or
`/reload`** — a hard limitation of the WoW client, not something this tool
can work around. To get updates without ever forcing a reload, this tool
combines two sources:

| Source | What it gives us | When it updates |
|---|---|---|
| **SavedVariables** (`RCLootCouncilLootDB`, inside `RCLootCouncil.lua`) | Structured fields: item, boss, instance, votes, response — no text parsing | Only on a *natural* logout/reload (end of raid, zoning, etc.) |
| **Chat log** (`WoWChatLog.txt`) | The award-announcement chat line, parsed via regex | Live, the instant the message is sent — but requires `/run LoggingChat(1)` |

`rclootparser watch` runs both: the chat log gives live updates, and the
SavedVariables pass reconciles/backfills the richer fields whenever a reload
happens anyway. Awards are deduped by content hash (item + winner + boss +
timestamp), so both sources reporting the same award is harmless.

## Setup

1. **Enable chat logging in-game** (only needed for the live path). This is
   a runtime toggle, not a saved setting — re-run it every session (a macro
   or an addon calling this on `PLAYER_LOGIN` avoids doing it by hand):
   ```
   /run LoggingChat(1)
   ```
2. **Build**:
   ```bash
   go build -o rclootparser .
   ```
3. **Configure**:
   ```bash
   ./rclootparser init
   ```
   Edit the generated `config.json`:
   - `saved_variables_path` — full path to `RCLootCouncil.lua` (**not**
     `RCLootCouncilLootDB.lua` — that's not a real file; WoW writes every
     SavedVariable an addon declares into one file named after the addon
     folder, so `RCLootCouncilLootDB` is a second assignment inside
     `RCLootCouncil.lua`, alongside `RCLootCouncilDB`) under
     `WTF/Account/<ACCOUNT>/SavedVariables/` in your WoW install
   - `chat_log_path` — full path to `Logs/WoWChatLog.txt` in your WoW
     install; observed behavior is that WoW appends to a single log file
     across sessions rather than rotating (confirmed for `WoWCombatLog.txt`
     in the same folder — chat logging is presumed to work the same way)
   - `api_base_url` — your deployed RCLootCouncilApi URL
   - `api_key` — the `INGEST_API_KEY` you set on that API (see its README)
4. **Run**:
   ```bash
   ./rclootparser sync-once   # one-shot: parse SavedVariables now, sync new awards, exit
   ./rclootparser watch       # continuous: live chat tail + periodic reconciliation
   ```

## How award text is matched

The chat-log matcher looks for RCLootCouncil's default announcement:
`"<player> was awarded with <item link> for <reason>!"`. If your guild
customized the `awardText` setting in the addon, override the pattern via
`award_announce_pattern` in `config.json` — it must be a Go regexp with
named groups `player`, `item`, and `reason`. See
[`internal/chatlog/chatlog.go`](internal/chatlog/chatlog.go) for the default.

## Known limitations

- **Timestamps are the game client's local wall-clock time**, not true
  UTC — SavedVariables and chat log don't carry a timezone.
- **Chat log rotation is unconfirmed**: this tool tails a single fixed path
  and assumes WoW appends to it indefinitely, based on observing
  `WoWCombatLog.txt` behave that way in the same install — not yet verified
  against a real `WoWChatLog.txt`, since `/run LoggingChat(1)` needs to
  actually produce one first. If it does turn out to rotate, this tool
  would need a `--watch-dir` mode to auto-discover the current file instead.
- **Single machine**: designed to run on one person's WoW install (e.g. the
  loot master), not to be run by every raider — running it on multiple
  machines for the same raid would report the same awards multiple times
  (deduped per-machine, but not across machines).

## Architecture

```
main.go                       CLI entry (init / sync-once / watch)
internal/config/              config.json loading
internal/luatable/            minimal Lua table-literal parser (no Lua VM)
internal/savedvars/           RCLootCouncil.lua -> []model.Award (+ []model.Player)
internal/chatlog/             chat log tailer + award-line regex matcher
internal/itemlink/            decodes |Hitem:...|h[Name]|h chat links
internal/model/               shared Award type + dedupe key
internal/state/                local sync progress (offsets, dedupe set)
internal/apiclient/           POST /api/loot client
internal/sync/                wires the above together
```

No external dependencies — standard library only, so `go build` works
offline. This is deliberately CLI-first; a tray-icon/GUI shell (e.g. via
[Wails](https://wails.io)) can wrap this same core once it's proven out
in daily use.

## Releases

A merge to `main` (which, since `main` is branch-protected, means it's
already passed CI) that touches actual code (`*.go`, `go.mod`, `go.sum`)
automatically publishes a new GitHub Release:
[`.github/workflows/release.yml`](.github/workflows/release.yml)
auto-increments the patch version from the latest `vX.Y.Z` tag, cross-compiles
`rclootparser` for Windows (amd64) and macOS (amd64 + Apple Silicon), and
attaches the binaries — no Go installation needed on the receiving end. A
merge that only touches docs/config/workflow files is skipped, so trivial
changes don't spam the Releases page.

To cut a specific version yourself instead (e.g. a deliberate major/minor
bump), push a tag directly — it takes priority over the auto-increment:
```bash
git tag v1.0.0
git push origin v1.0.0
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for local setup and PR guidelines.

## Security

Found a vulnerability? See [SECURITY.md](SECURITY.md) for how to report it privately.

## License

[GPL-3.0](LICENSE)
