# autoGit

A macOS-first CLI that auto-commits your Git work **on change**, **on a schedule**, or **both**, with:

- Interactive setup wizard (`--setup`)
- Interval commits and on-change commits with **batching** (window + idle)
- `.gitignore` parsing merged with custom excludes
- Signed commits (`-S`) and message **trailers** (e.g., `Co-authored-by`)
- **Multi-repo** support
- Human-readable logs with **rotation** (default: `~/Library/Logs/autoGit.log`)
- Themed, colored console output (auto/dark/light/mono)
- Optional **LaunchAgent** to run at login

## Install

```bash
# requires Go 1.22+
./install.sh
# or build manually
go mod tidy && go build -o autoGit ./cmd/autoGit
```

## Quick start

```bash
# first-time wizard
./autoGit --setup

# from then on (uses saved config)
./autoGit

# override theme for a run
./autoGit --theme mono
```

## Config

Default path: `~/.config/autoGit/config.yaml` (override via `GITAUTOCOMMIT_CONFIG`).

See [`examples/config.example.yaml`](examples/config.example.yaml) for all fields.

## LaunchAgent

Wizard can write & load a plist at `~/Library/LaunchAgents/com.gitautocommit.cli.plist`.

## Uninstall

- Remove plist: `launchctl remove com.gitautocommit.cli && rm ~/Library/LaunchAgents/com.gitautocommit.cli.plist`
- Remove binary: `sudo rm /usr/local/bin/autoGit`
- Remove logs/config if desired.

## Notes

- History hygiene: consider committing to an `autosave` branch and merging selectively.
- CI safety: if your remote triggers CI on push, either disable `push` or increase `interval`.
- `.gitignore` parser here is line-based (simple). For exotic patterns, add a robust matcher.