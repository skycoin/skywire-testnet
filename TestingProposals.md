# Testing Proposals

- [Testing Proposals](#testing-proposals)
	- [Foreword](#foreword)
	- [1. Tests in development environment and automated tests](#1-tests-in-development-environment-and-automated-tests)
		- [Description](#description)
		- [Current status in skywire](#current-status-in-skywire)
		- [Current status in other skycoin projects (only about skycoin/skycoin now)](#current-status-in-other-skycoin-projects-only-about-skycoinskycoin-now)
		- [Tools](#tools)
		- [Possible proposals](#possible-proposals)
	- [2. "Blackbox"-"Whitebox"-other "boxes". Integration tests, load tests, etc](#2-%22blackbox%22-%22whitebox%22-other-%22boxes%22-integration-tests-load-tests-etc)
		- [Descriptions](#descriptions)
		- [Current status](#current-status)
		- [Tools](#tools-1)
		- [Proposals](#proposals)
	- [4. Load testing, Benchmarks](#4-load-testing-benchmarks)
		- [Description](#description-1)
		- [Current status in skywire](#current-status-in-skywire-1)
		- [Tools](#tools-2)
		- [Proposals](#proposals-1)
	- [5. Travis CI-tests](#5-travis-ci-tests)
		- [Description](#description-2)
		- [Current status](#current-status-1)
		- [Tools](#tools-3)
		- [Proposals](#proposals-2)
	- [6. Fuzzy-testing. go-fuzz](#6-fuzzy-testing-go-fuzz)
	- [Description](#description-3)
		- [Current status in skycoin/skywire](#current-status-in-skycoinskywire)
		- [Current status in skycoin/skycoin](#current-status-in-skycoinskycoin)
		- [Tools](#tools-4)
		- [Proposals](#proposals-3)
	- [7. Non-CI tests](#7-non-ci-tests)
		- [Description](#description-4)
		- [Current status in skywire](#current-status-in-skywire-2)
		- [Current status in other skycoin projects](#current-status-in-other-skycoin-projects)
		- [Tools](#tools-5)
		- [Proposals](#proposals-4)
	- [Roadmap](#roadmap)

## Foreword

This document is not an attempt to write a monography on software testing.

It's a living document summarizing themes brought up in discussions.

**Goals** are to describe:

- the current status of our testing: What tests we are running, how they are running, where they are running
- long term proposals for the testing: What tests, how and where.
- create a small „roadmap“: What is most urgent? What can be done later?

**Non-goals**:

- absolute correctness
- absolute logical consistency
- full coverage of the theme of software testing

-------------------------------------------------------------

## 1.  Tests in  development environment and automated tests

### Description

**Developer environment**:

- very simplistic but allows fast refresh-restart
- it could be 2-3 nodes  in one-two networks
- easily integrated with IDEs, etc
- allows manual intervention, e.g. Ctrl-C, edit config, start again
- each developer creates their own developer environment according to their tastes. So it's more recipes than a hard-defined

**Test-environment(s)**:

- it could be tens (and hundreds) nodes with complex network topologies, different switches between them, different speeds
and we don't have requirements for test-environment yet
- mostly automated
- some are runnable by CIs
- some are not suitable for CIs

### Current status in skywire

**A. Test-environment**

We have automated test-environment for Travis-CI, runnable with `make test` manually

```
test: ## Run tests for net
        ${OPTS} go test -race -tags no_ci -cover -timeout=5m ./internal/...
        ${OPTS} go test -race -tags no_ci -cover -timeout=5m ./pkg/...
```

We have coverage ranging from ~40% to ~90% for v

**B. Developer-environment**

- `make test` from Makefile 
- We started some recipes for developers in README.md
- Created `make-goals` for running nodes: `make run`, `make stop`, `make docker-run`, `make docker-stop`

### Current status in other skycoin projects (only about skycoin/skycoin now)

In skycoin/skycoin:

**A. Daemonization**

They have possibilities to run in daemonized/detached modes.
It could be good for .. e.g. `make run-daemon` && tail -f [some_log_file]`
```
run-client                     Run skycoin with desktop client configuration. To add arguments, do 'make ARGS="--foo" run'.
run-daemon                     Run skycoin with server daemon configuration. To add arguments, do 'make ARGS="--foo" run'.
```

Added daemonization even in skycoin/scycoin itself
.. add more later ..

**B. A lot of "integration"-goals**


```sh
integration-test-stable        Run stable integration tests
integration-test-stable-disable-header-check Run stable integration tests with header check disabled
integration-test-stable-disable-csrf Run stable integration tests with CSRF disabled
integration-test-live          Run live integration tests
integration-test-live-wallet   Run live integration tests with wallet
integration-test-live-enable-header-check Run live integration tests against a node with header check enabled
integration-test-live-disable-csrf Run live integration tests against a node with CSRF disabled
integration-test-live-disable-networking Run live integration tests against a node with networking disabled (requires wallet)
integration-test-disable-wallet-api Run disable wallet api integration tests
integration-test-enable-seed-api Run enable seed api integration test
integration-test-disable-gui   Run tests with the GUI disabled
integration-test-db-no-unconfirmed Run stable tests against the stable database that has no unconfirmed transactions
integration-test-auth          Run stable tests with HTTP Basic auth enabled
```

Live-integration-tests are described in [README.md#live-integration-tests](https://github.com/skycoin/skycoin/blob/develop/README.md#live-integration-tests).

Those goals are implemented (mostly from what I saw) as bash-scripts in `./ci_scripts` folder (..need to check later..)

For dockerized environments there are `./docker` folder (we have now just one `skywire-runner.Dockerfile` )

.. it's needed to dive in more in skycoin/skycoin tests ...

**C. Fuzzy testing**

```text
fuzz-base58: ## Fuzz the base58 package. Requires https://github.com/dvyukov/go-fuzz
	go-fuzz-build github.com/skycoin/skycoin/src/cipher/base58/internal
	go-fuzz -bin=base58fuzz-fuzz.zip -workdir=src/cipher/base58/internal

fuzz-encoder: ## Fuzz the encoder package. Requires https://github.com/dvyukov/go-fuzz
	go-fuzz-build github.com/skycoin/skycoin/src/cipher/encoder/internal
	go-fuzz -bin=encoderfuzz-fuzz.zip -workdir=src/cipher/encoder/internal
```

.. need to dive in more how and why they do `go-fuzz` ..

### Tools

### Possible proposals

**For testing environments**:

- ci_scripts for integration tests
- more blackbox-tests, we have only whitebox right now
- fuzzy tests
- .. need more ...

**For developer environment specifically**:

- more recipes later, maybe including OS-specific/IDE-specific recipes
- `detached`/`daemonized` modes for running nodes
- more commands in `skywire-cli` such as `pk` proposed by Evan
- .. more ..

-------------------------------------------------------------------

## 2. "Blackbox"-"Whitebox"-other "boxes". Integration tests, load tests, etc

### Descriptions

**Whitebox**: everything is known about internals

**Blackbox**: nothing is known about internals

**Integration tests**: ["The point of integration testing, as the name suggests, is to test whether many separately developed modules work together as expected" M. Fowler](https://martinfowler.com/bliki/IntegrationTest.html)

**Load tests**: .. describe later ..

**Benchmarks**: .. later ..

**Quikcheck**-style tests: [generating test cases for test suites](https://en.wikipedia.org/wiki/QuickCheck)

**Fuzzy tests**: (https://en.wikipedia.org/wiki/Fuzzing)  it does not know about internals but it analyzes outputs to find breaking inputs. It's a kind of Quikcheck

Styles:

doctest/Example style ..

### Current status

```bash
$ find pkg -name "*test.go" |wc -l
# 34
$ find internal -name "*test.go" |wc -l
# 12
$ find cmd -name "*test.go" |wc -l
# 0
```

Our tests are:

- whitebox
- only one Example is found  

### Tools

### Proposals

-------------------------------------------------------------------

## 4. Load testing, Benchmarks

### Description

.. write short descriptions ..

### Current status in skywire

.. need to dive more, from what I see: we don't have them yet ..

### Tools

### Proposals

-------------------------------------------------------------------

## 5. Travis CI-tests

### Description

Travis-CI tests are runned for each PR.

CI-tests must have duration not exceeding some value.

From discussion we came to: "Not more than 30 minutes"

### Current status

We have Travis-CI tests runned by `make test`

Durations are between 2-3 minutes

We don't use Travis to run dockerized tests

### Tools

### Proposals

... need to dive more into other skycoin projects ...

-------------------------------------------------------------------

## 6. Fuzzy-testing. go-fuzz

## Description

Note:
    Take my words with tonns of salt: I never did fuzzy-tests myself.
    I'm quoting right now from memory Dmitry Viukov

Fuzzy tests can be described as a kind of Quikcheck-tests.
But they are not "completely blackbox" - they use outputs to find breaking inputs.

There is some interesting mathematical theory used in them.

If I remember correctly `go-fuzz` itself is a floating number generator.

How it works:
- we have n-parameter function
- so we have an n-dimensional space of possible params for this function
- go-fuzz generates a big floating number
- and this generated number is mapped to point in n-dimensional space of params
- go-fuzz executes function with generated params
- analyzes output. In case of exceptions/panics/errors - it stores result
- and error-outputs are used to generate next possible "breaking inputs"

So it's a sort of "guided Quikcheck": instead of completely random jumps
it specifically tries to find "points of maximum pain".

As far as I remember they are especially good for:

- usually finds breaking inputs faster than "naive-Quikcheck"s
- find errors in "crossing borders": borders between code and dynamic linked libraries, switching OS contexts.
  And in networks we have "crossing borders" everywhere

### Current status in skycoin/skywire

We don't have them.
And maybe it's prematurely to start create them.
But they could/must be useful for skywire in future.

### Current status in skycoin/skycoin

```make
fuzz-base58: ## Fuzz the base58 package. Requires https://github.com/dvyukov/go-fuzz
	go-fuzz-build github.com/skycoin/skycoin/src/cipher/base58/internal
	go-fuzz -bin=base58fuzz-fuzz.zip -workdir=src/cipher/base58/internal

fuzz-encoder: ## Fuzz the encoder package. Requires https://github.com/dvyukov/go-fuzz
	go-fuzz-build github.com/skycoin/skycoin/src/cipher/encoder/internal
	go-fuzz -bin=encoderfuzz-fuzz.zip -workdir=src/cipher/encoder/internal
```

.. need to analyze more ..

### Tools

### Proposals

I'm not sure that today go-fuzz will be beneficial for us.
I think we need more other types of tests today.
And when everything will start to work - start fuzzy tests

-------------------------------------------------------------------

## 7. Non-CI tests

### Description

Non-CI tests could be:

- they are too long - and breaks development workflow
- they use too much resources to create test environment  
- they are impossible to run from CI by another reason

... must check what travis-ci could do ...

### Current status in skywire

We don't have them yet
But the need should/must arise in future

### Current status in other skycoin projects

.. need to dive more ..

### Tools

### Proposals

------------------------------------------------------------------

## Roadmap

TBD