#!/bin/zsh
## This script working incorrect in bash and sh because of `/**/*`-patterns in it 
## Sorry. Anyway it's a prototype of sorts. It's  a 'throwaway-script' now
echo pkg:
echo
echo 'Test*': 'go test': $(go test ./pkg/... -list Test |rg Test --count), 'naive rg': $(rg Test ./pkg/**/*.go --json |jq -r ".data.stats.matches" |tail -1)
echo 'Example*': 'go test': $(go test ./pkg/... -list Example |rg Example --count), 'naive rg': $(rg Example ./pkg/**/*.go --json |jq -r ".data.stats.matches" |tail -1)
echo 'Benchmark*': 'go test': $(go test ./pkg/... -list Benchmark |rg Benchmark --count), 'naive rg': $(rg Benchmark ./pkg/**/*.go --json |jq -r ".data.stats.matches" |tail -1)
echo 'Integration. go test-able': $(rg integration  ./pkg/**/*test.go  --json |jq -r ".data.stats.matches" |tail -1)
echo 'No-CI': $(rg "no_ci" ./pkg/**/*.go --json |jq -r ".data.stats.matches" |tail -1)
echo 'go-fuzz': Not implemented

echo
echo internal:
echo
echo 'Test*': 'go test': $(go test ./internal/... -list Test |rg Test --count), 'naive rg': $(rg Test ./internal/**/*.go --json |jq -r ".data.stats.matches" |tail -1)
echo 'Example*': 'go test': $(go test ./internal/... -list Example |rg Example --count), 'naive rg': $(rg Example ./internal/**/*.go --json |jq -r ".data.stats.matches" |tail -1)
echo 'Benchmark*': 'go test': $(go test ./internal/... -list Benchmark |rg Benchmark --count), 'naive rg': $(rg Benchmark ./internal/**/*.go --json |jq -r ".data.stats.matches" |tail -1)
echo 'Integration. go test-able': $(rg integration  ./internal/**/*test.go  --json |jq -r ".data.stats.matches" |tail -1)
echo 'No-CI': $(rg "no_ci" ./internal/**/*.go --json |jq -r ".data.stats.matches" |tail -1)
echo 'go-fuzz': Not implemented 

echo
echo cmd:
echo
echo 'Test*': 'go test': $(go test ./cmd/... -list Test |rg Test --count), 'naive rg': $(rg Test ./cmd/**/*.go --json |jq -r ".data.stats.matches" |tail -1)
echo 'Example*': 'go test': $(go test ./cmd/... -list Example |rg Example --count), 'naive rg': $(rg Example ./cmd/**/*.go --json |jq -r ".data.stats.matches" |tail -1)
echo 'Benchmark*': 'go test': $(go test ./cmd/... -list Benchmark |rg Benchmark --count), 'naive rg': $(rg Benchmark ./cmd/**/*.go --json |jq -r ".data.stats.matches" |tail -1)
echo 'Integration. go test-able': $(rg integration  ./cmd/**/*test.go  --json |jq -r ".data.stats.matches" |tail -1)
echo 'No-CI': $(rg "no_ci" ./cmd/**/*.go --json |jq -r ".data.stats.matches" |tail -1)
echo 'go-fuzz': Not implemented 

