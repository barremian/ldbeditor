#!/usr/bin/env bash
set -euo pipefail

WAILS_VERSION="$(awk '/github.com\/wailsapp\/wails\/v3 / { print $2; exit }' go.mod)"

sudo env "PATH=$PATH" GOBIN=/usr/local/bin go install github.com/go-task/task/v3/cmd/task@latest
sudo env "PATH=$PATH" GOBIN=/usr/local/bin go install "github.com/wailsapp/wails/v3/cmd/wails3@${WAILS_VERSION}"

go mod download
npm ci --prefix frontend
