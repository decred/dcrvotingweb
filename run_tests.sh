#!/usr/bin/env bash

set -ex

# run tests
env GORACE="halt_on_error=1" go test -race ./...

# run linters when not running as a GitHub action
[ -z "$GITHUB_ACTIONS" ] && source ./lint.sh

echo "------------------------------------------"
echo "Tests completed successfully!"
