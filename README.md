# snapshot

your filesystem's safety net. automatic periodic backups that protect your workspaces from destructive ai agents, bad refactors, and accidental `rm -rf` moments.

## why

ai coding agents (cursor, claude code, copilot, etc.) can rewrite, delete, or corrupt files faster than you can review them. one wrong tool call and your afternoon is gone.

snapshot runs in the background, silently creating incremental backups of your workspaces every few minutes. when something goes wrong, you roll back in seconds.

- **automatic** - set it once, backups run on a schedule (every 10m, 30m, 1h, or 6h)
- **incremental** - only changes are stored, so backups are fast and tiny
- **encrypted** - every backup repo is encrypted at rest
- **beautiful tui** - browse, restore, and diff snapshots interactively

## install

```sh
curl -fsSL https://raw.githubusercontent.com/sunneydev/snapshot/main/install.sh | bash
```

or with homebrew:

```sh
brew install sunneydev/tap/snapshot
```

## get started

```sh
snapshot add ~/work
```

that's it. you'll be prompted to enable automatic backups right away. pick an interval and forget about it.

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

## automatic backups

the whole point. once enabled, snapshot creates backups on a schedule using launchd (macos) or cron (linux).

```sh
snapshot auto on 30m    # every 30 minutes
snapshot auto           # check status
snapshot auto off       # disable
```

intervals: `10m`, `30m`, `1h`, `6h`

you can also enable this when adding a workspace with `snapshot add`.

## commands

| command | what it does |
|---------|-------------|
| `snapshot` | open the tui |
| `snapshot save [workspace]` | create a snapshot now |
| `snapshot list [workspace]` | list snapshots |
| `snapshot restore <path> [id]` | restore a file |
| `snapshot diff <path>` | diff a file against the last snapshot |
| `snapshot add <path>` | register a workspace |
| `snapshot rm <path>` | unregister a workspace |
| `snapshot ws` | list workspaces |
| `snapshot auto [on\|off] [interval]` | manage automatic backups |

## how it works

each workspace gets its own encrypted backup repository with automatic deduplication. backups are incremental so only changes since the last snapshot are stored.

large files (>20MB), build artifacts, node_modules, .git, and other common noise are excluded by default.

## configuration

config lives in `~/.config/snapshot/` (or `$XDG_CONFIG_HOME/snapshot/`).

| file | purpose |
|------|---------|
| `workspaces` | registered workspace paths |
| `password` | encryption password |
| `schedule` | auto-backup interval |

backup repos are stored in `~/.local/share/snapshot/repos/`.

## claude code integration

snapshot works as a `/snapshot` skill in [claude code](https://claude.ai/claude-code), so you can save and restore snapshots before risky operations without leaving your terminal.

## license

[MIT](LICENSE)
