# "Periodic Table of Tests"

## Description

### 1. Tests as `Test*`-functions

Description:

    Test as golang `Test*`-function runnable by `go test`

Motivation:

    Part of development process.
    Why/when needed: to ensure that written code works in principle.

Measurement:

    a. with `go test`:  `go test ./[path]/... -list Test |rg Test --count`
    b. naive `rg`: `rg Test ./[path]/**/*.go --json |jq -r ".data.stats.matches" |tail -1`

    Methods can/should give different results.
    Because "naive rg" returns commented tests too.
    It should/could/must be a point of interest if they differ.

Coverage:

    Using `go test -cover`

### 2. Examples

Description:

    Test as golang  `Example*`-function runnable by `go test`
    Subtype of golang tests. Combines features of test and documentation.

Motivation:

    Part of development and documentation.

    When/why needed:
    Example-type test could a poor test itself.
    It's hard to create them table-driven.
    But in the same time it could be a great documentation.
    So decision "Write more Examples" could more political decision then technical.

Measure:

    a. with `go test`:  `go test ./[path]/... -list Example |rg Example --count`
    b. naive `rg`: `rg Example ./[path]/**/*.go --json |jq -r ".data.stats.matches" |tail -1`

    Methods can give different results.

    Because:
    - "naive rg" returns commented Examples too
    - `go test` filters not-compileable Examples: e.g. has no `Output: `, resides in incorrect place
  
    It should/could/must be a point of interest if they differ.

Coverage:

    Examples as golang tests:  measured by `go test -cover`
    Examples as documentation: measured by ratio "quantity of Examples" /"desired quantity of Examples"

### 3. Benchmarks

Description:

    Tests as `Benchmark*`-functions.

Motivation:

    To ensure performance.

Measure:

    a. with `go test`:  `go test ./[path]/... -list Benchmark |rg Benchmark --count`
    b. naive `rg`: `rg Benchmark ./[path]/**/*.go --json |jq -r ".data.stats.matches" |tail -1`

Coverage:

    TBD

### 4. Integration tests

Description:

    Test designed to ensure that modules of system works
    in integration with other modules and outside world.

    Subtype A: written as `go test`-able function. In most cases whitebox-style.
    Subtype B: written as `./ci_scripts` executable. In most cases blackbox-style.
                Could be written in any language/framework. 
                In `skycoin/skycoin` now: they are  `bash`-scripts, in theory could any executable
                In `skycoin/skywire`: We don't have them yet.

Motivation:

    To ensure that modules of system works:
      -  with each other
      -  with outside world

Measure:

    Subtype A: `rg integration  ./pkg/**/*test.go  --json |jq -r ".data.stats.matches" |tail -1`
    Subtype B: `ls ./ci_scripts |wc -l`

Coverage:

    Subtype A: with `go test -cover`
    Subtype B: must be measured as ratio of "implemented integration test cases"/"desired integration test cases"

### 5. No-CI tests

Description:

    Tests that not suitable to be runned by CI.

Motivation:

    - It could be not possible to run such tests by CI
    - Even when it's possible - long duration would break reviewing process

Measure:

    `rg "no_ci" ./[path]/**/*.go --json |jq -r ".data.stats.matches" |tail -1`

    Note:

        `skycoin/skycoin` use another style of switching on/off tests: tests are switched with ENV-vars.

        Method of measurement could be:
        `rg "enabled" ./[path]/**/*test.go --json |jq -r ".data.stats.matches" |tail -1`

Coverage:

    Not applicable

### 6. Fuzzy tests

Description:

    Tests that not suitable to be runned by CI.

Motivation:

    Ensure that there are no "holes" in implementation.
    "You cannot be sure if you don't fuzz it" D.Vyukov

Measurement:

    TBD

Coverage:

    ?

## Summary of motivations:  "Why/When we need a specific test type?"

1. `Test*`-type: in development process to ensure that "it works"
2. `Example*`-type: in addition to `Test*` we want it to be documented. E.g. to make more easy life of newcomers to project
3. `Benchmark`-type: to ensure performance
4. Integration: to ensure modules comminicates correctly
5. No-CI: to ensure that we don't break our review-process
6. Fuzzy: to ensure that there are "no holes" in implementation

## Test-types per package

pkg:

Test*: go test: 108, naive rg: 144
Example*: go test: , naive rg: 1
Benchmark*: go test: , naive rg: 0
Integration. go test-able: 6
go-fuzz: Not implemented

internal:

Test*: go test: 29, naive rg: 33
Example*: go test: , naive rg: 0
Benchmark*: go test: , naive rg: 0
Integration. go test-able: 0
go-fuzz: Not implemented

cmd:

Test*: go test: , naive rg: 0
Example*: go test: , naive rg: 0
Benchmark*: go test: , naive rg: 0
./period-table-of-tests.sh:27: no matches found: ./cmd/**/*test.go
Integration. go test-able:
go-fuzz: Not implemented

## Integration type. `./ci_scripts`

Total: 0/0 = NaN

We don't have:

- description of desired integration cases. Something like "Integration.md" in skycoin
- no `./ci_scripts`