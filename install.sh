#!/usr/bin/env bash
set -euo pipefail

APP="cc-connect"
INSTALL_DIR="${GOBIN:-${GOPATH:-$HOME/go}/bin}"
INSTALL_PATH="${INSTALL_DIR}/${APP}"

# Ensure Go is available
GO_BIN=""
for candidate in "$(command -v go 2>/dev/null)" "/usr/local/go/bin/go" "$HOME/go/bin/go" "$HOME/.local/go/bin/go"; do
    if [ -n "$candidate" ] && [ -x "$candidate" ]; then
        GO_BIN="$candidate"
        break
    fi
done

if [ -z "$GO_BIN" ]; then
    echo "Error: Go not found. Please install Go first: https://go.dev/dl/"
    exit 1
fi

export PATH="$(dirname "$GO_BIN"):$PATH"

echo "Using Go: $(go version)"
echo "Install path: ${INSTALL_PATH}"

# Build from source
cd "$(dirname "$0")"

VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo "dev")"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo "none")"
BUILD_TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}"

# Build web frontend if needed
if [ -d web ] && [ ! -d web/dist ]; then
    echo "Building web frontend..."
    if command -v npm &>/dev/null; then
        (cd web && [ -d node_modules ] || npm install && npm run build)
    else
        echo "Warning: npm not found, building without web UI (no_web tag)"
        EXTRA_TAGS="no_web"
    fi
fi

echo "Building ${APP} ${VERSION}..."
go build ${EXTRA_TAGS:+-tags "$EXTRA_TAGS"} -ldflags "${LDFLAGS}" -o "${INSTALL_PATH}" ./cmd/cc-connect

echo ""
echo "Installed: ${INSTALL_PATH}"
"${INSTALL_PATH}" --version
