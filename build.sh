#!/usr/bin/env bash
set -euo pipefail

echo "Building rig..."
mkdir -p output
go build -o output/rig src/main.go

echo "Packaging distribution files into output/..."
cp config.json output/
cp instructions.md output/

echo "Done! Distribution ready in output/:"
ls -lh output/
