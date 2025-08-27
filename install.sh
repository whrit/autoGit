#!/usr/bin/env bash
set -euo pipefail

if ! command -v go >/dev/null 2>&1; then
  echo "Go 1.22+ is required" >&2
  exit 1
fi

echo "==> Building autoGit"
GOFLAGS=${GOFLAGS:-}
GO111MODULE=on go mod tidy
GO111MODULE=on go build -o autoGit ./cmd/autoGit

BIN="/usr/local/bin/autoGit"
echo "==> Installing to $BIN (sudo)"
sudo mv -f autoGit "$BIN"

CFG_DIR="$HOME/.config/autoGit"
CFG="$CFG_DIR/config.yaml"
mkdir -p "$CFG_DIR"
if [ ! -f "$CFG" ]; then
  cat > "$CFG" <<'YAML'
# minimal default; run `autoGit --setup` to customize
theme: auto
log_path: "$HOME/Library/Logs/autoGit.log"
repos:
  - path: "."
    watch: true
    interval: 20m
    parse_gitignore: true
YAML
  echo "Wrote default config → $CFG"
fi

read -r -p "Create LaunchAgent to run at login? [y/N] " ans
if [[ "$ans" =~ ^[Yy]$ ]]; then
  "$BIN" --setup
fi

echo "Done. Try: autoGit --setup"