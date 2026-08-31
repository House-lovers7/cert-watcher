#!/usr/bin/env bash
# Windows 向けバイナリをクロスコンパイルし、会社PCへ持ち込む配布zipを作る。
# 使い方: scripts/package-release.sh v0.2.0
# 出力:   dist/cert-watcher-<version>-windows-amd64.zip
set -euo pipefail

version="${1:?usage: scripts/package-release.sh <version, e.g. v0.2.0>}"
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

name="cert-watcher-${version}-windows-amd64"
stage="dist/${name}"

rm -rf "$stage" "dist/${name}.zip"
mkdir -p "$stage/config" "$stage/scripts"

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-s -w" -o "$stage/cert-watcher.exe" ./cmd/cert-watcher

cp README.md LICENSE INSTALL-windows.md "$stage/"
cp config/sites.csv config/targets.csv "$stage/config/"
cp config/settings.example.json config/origins.secure.example.csv config/action_status.example.csv "$stage/config/"
cp scripts/register-task.ps1 "$stage/scripts/"

# macOS 由来のゴミを同梱しない（zipはFinderではなくCLIで作る）
find "$stage" -name '.DS_Store' -delete
(cd dist && zip -qrX "${name}.zip" "$name" -x '*.DS_Store')

echo "built: dist/${name}.zip"
unzip -l "dist/${name}.zip"
