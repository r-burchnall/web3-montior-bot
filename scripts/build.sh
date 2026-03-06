#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
AGENT_DIR="$PROJECT_ROOT/fcsc-agent"
OUTPUT_DIR="$PROJECT_ROOT/build"

GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"

mkdir -p "$OUTPUT_DIR"

echo "Building fcsc-agent for ${GOOS}/${GOARCH}..."
cd "$AGENT_DIR"

CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
    -ldflags="-s -w" \
    -o "$OUTPUT_DIR/fcsc-agent" \
    ./cmd/fcsc-agent/

echo "Built: $OUTPUT_DIR/fcsc-agent"
ls -lh "$OUTPUT_DIR/fcsc-agent"
