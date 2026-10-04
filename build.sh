#!/usr/bin/env bash
set -euo pipefail

echo "Cleaning output/..."
mkdir -p output
rm -rf output/*

echo "Building rig..."
go build -o output/rig src/main.go

echo "Packaging distribution files into output/..."
cp config/config.json output/
cp config/instructions.md output/

echo "Done!"
ls -lh output/

echo "To run: cd output && ./rig"
