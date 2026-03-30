# snapshot

automatic workspace backups with a beautiful tui.

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

each workspace gets its own backup repository with automatic deduplication and encryption. large files (>20M), build artifacts, node_modules, .git, and other common junk are excluded by default.

backups are incremental - only changes since the last snapshot are stored, so they're fast and space-efficient.

## configuration

all config lives in `~/.config/snapshot/` (or `$XDG_CONFIG_HOME/snapshot/`).

| file | purpose |
|------|---------|
| `workspaces` | registered workspace paths, one per line |
| `password` | repository encryption password |

data (repos) is stored in `~/.local/share/snapshot/repos/` (or `$XDG_DATA_HOME/snapshot/repos/`).

### default excludes

node_modules, .git, dist, build, .next, .turbo, .gradle, .venv, __pycache__, .cache, coverage, and more. files larger than 20M are skipped.

## automatic backups

snapshot can set up automatic backups for you (launchd on macOS, cron on linux).

```sh
snapshot auto on 30m
```

you'll also be asked during `snapshot add` if you want to enable this. options: `10m`, `30m`, `1h`, `6h`.

```sh
snapshot auto          # check status
snapshot auto off      # disable
```

## claude code integration

snapshot works as a `/snapshot` skill in [claude code](https://claude.ai/claude-code). use it to save and restore snapshots before risky operations.

## requirements

- [restic](https://restic.net) - installed automatically with homebrew, or `apt install restic` on linux

## license

[MIT](LICENSE)
