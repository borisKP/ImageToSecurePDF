#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APP="$ROOT/macos/ImageToSecurePDF.app"
MACOS="$APP/Contents/MacOS"
GO="${GO:-go}"

mkdir -p "$MACOS"

# 版本号唯一来源：根目录 VERSION
VER="$(cat "$ROOT/VERSION" | tr -d '\r\n')"
plutil -replace CFBundleShortVersionString -string "$VER" "$APP/Contents/Info.plist"
plutil -replace CFBundleVersion -string "$VER" "$APP/Contents/Info.plist"

# 共享 Go 引擎
(cd "$ROOT/engine" && "$GO" build -o "$MACOS/imgtopdf-helper" .)

# SwiftUI 前端
swiftc -parse-as-library -O "$ROOT/macos/App.swift" -o "$MACOS/ImageToSecurePDF"

echo "Built $APP ($VER)"
