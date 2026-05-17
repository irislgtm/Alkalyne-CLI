#!/bin/sh
set -e
export PATH="$HOME/go/bin:$PATH"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

pass() { printf " ${GREEN}PASS${NC}\n"; }
fail() { printf " ${RED}FAIL${NC}\n"; exit 1; }
check() {
	printf "%-50s" "$1"
	shift
	if "$@" 2>/dev/null; then pass; else fail; fi
}

check "gofmt"		gofmt -l -e .
check "go vet"		go vet ./...
check "go vet race"	go vet -race ./...
check "import boundaries"	go test ./internal/archtest/...
check "unit tests"		go test ./internal/...
check "go mod tidy"		go mod tidy && git diff --exit-code go.mod go.sum
check "golangci-lint"	golangci-lint run ./...
