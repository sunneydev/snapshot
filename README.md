# snapshot

workspace backup tool powered by restic with a beautiful tui.

## install

```sh
npm i -g snapshot-backup
```

```sh
brew install sunneydev/tap/snapshot
```

```sh
go install github.com/sunneydev/snapshot@latest
```

or download a binary from [releases](https://github.com/sunneydev/snapshot/releases).

## demo

```
  snapshot
  ~/work

  > save            create a snapshot
    list            browse snapshots
    restore         recover a file
    diff            compare with snapshot
    workspaces      manage workspaces

  ↑/↓ navigate · enter select · q quit
```

## quick start

```sh
snapshot add ~/work     # register a workspace
snapshot save           # create your first snapshot
snapshot                # open the tui
```

## commands

| command | description |
|---------|-------------|
| `snapshot` | open the interactive tui |
| `snapshot save [path]` | create a snapshot of a workspace |
| `snapshot list [path]` | list snapshots for a workspace |
| `snapshot restore <path> [id]` | restore a file from a snapshot |
| `snapshot diff <path>` | diff a file against the latest snapshot |
| `snapshot add <path>` | register a workspace |
| `snapshot rm <path>` | unregister a workspace |
| `snapshot ws` | list registered workspaces |

## how it works

snapshot uses [restic](https://restic.net) under the hood. each workspace gets its own restic repository with automatic deduplication. large files (>20M), build artifacts, node_modules, .git, and other common junk are excluded by default.

backups are fast because restic only stores what changed since the last snapshot.

## configuration

all config lives in `~/.config/snapshot/` (or `$XDG_CONFIG_HOME/snapshot/`).

| file | purpose |
|------|---------|
| `workspaces` | registered workspace paths, one per line |
| `password` | restic repository password |

data (repos) is stored in `~/.local/share/snapshot/repos/` (or `$XDG_DATA_HOME/snapshot/repos/`).

### default excludes

node_modules, .git, dist, build, .next, .turbo, .gradle, .venv, __pycache__, .cache, coverage, and more. files larger than 20M are skipped. see `restic.go` for the full list.

## automatic backups

### macos (launchd)

save this as `~/Library/LaunchAgents/dev.sunney.snapshot.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>dev.sunney.snapshot</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/snapshot</string>
        <string>save</string>
    </array>
    <key>StartInterval</key>
    <integer>1800</integer>
    <key>StandardOutPath</key>
    <string>/tmp/snapshot.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/snapshot.log</string>
</dict>
</plist>
```

```sh
launchctl load ~/Library/LaunchAgents/dev.sunney.snapshot.plist
```

### linux (cron)

```sh
# every 30 minutes
*/30 * * * * /usr/local/bin/snapshot save >> /tmp/snapshot.log 2>&1
```

## claude code integration

snapshot works as a `/snapshot` skill in [claude code](https://claude.ai/claude-code). use it to save and restore snapshots before risky operations.

## requirements

- [restic](https://restic.net) (`brew install restic` / `apt install restic`)

## license

[MIT](LICENSE)
