#!/bin/sh
# Fresh machine / Claude Code cloud session: tools, deps, wasm build, tests.
set -e
cd "$(dirname "$0")/.."
command -v go >/dev/null || { echo "Go 1.23+ is required (https://go.dev/dl)"; exit 1; }
go version
if ! node -e 'process.exit(+process.versions.node.split(".")[0] >= 22 ? 0 : 1)' 2>/dev/null; then
	[ -s "$HOME/.nvm/nvm.sh" ] && . "$HOME/.nvm/nvm.sh" && (nvm use 24 >/dev/null 2>&1 || nvm install 24)
fi
node --version
npm ci
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o public/wasm/qud.wasm ./cmd/wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" public/wasm/ 2>/dev/null || cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" public/wasm/
go vet ./engine
go test ./engine
echo "ready: npm run dev (port 5195)"
