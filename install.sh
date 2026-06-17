#!/usr/bin/env bash
set -e
DIR="$(cd "$(dirname "$0")" && pwd)"
echo "Building GitEfFive..."
go build -o "$DIR/GitEfFive" "$DIR"
echo "Installing to /usr/local/bin/GitEfFive..."
sudo cp "$DIR/GitEfFive" /usr/local/bin/GitEfFive
echo "Done. Run 'GitEfFive' from anywhere."
