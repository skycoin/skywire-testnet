# Tests. Detection of unstable tests

## Synopsis

This document describes a procedure to detect tests that FAIL with some probability, e.g. 10-15% of runs.

Such tests are questionable themselves and they add instability of CI builds.

## Step-by-step procedure

### 1. Create list of tests

Use:

```bash
go test ./pkg/... -list "Test*|Example*" > ./logs/list-of-pkg-tests.txt  # use another path or filename if you wish
go test ./internal/... -list "Test*|Example*" > ./logs/list-of-internal-tests.txt
```

You will get output similar to:

```text
TestClient
ok  	github.com/SkycoinProject/skywire-mainnet/internal/httpauth	0.043s
?   	github.com/SkycoinProject/skywire-mainnet/internal/httputil	[no test files]
TestAckReadWriter
TestAckReadWriterCRCFailure
TestAckReadWriterFlushOnClose
TestAckReadWriterPartialRead
TestAckReadWriterReadError
TestLenReadWriter
ok  	github.com/SkycoinProject/skywire-mainnet/internal/ioutil	0.049s
```

Filter lines with `[no test files]`.

Transform this output to:

```bash
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/SkycoinProject/skywire-mainnet/internal/httpauth -run TestClient >> ./logs/internal/TestClient.log

go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/SkycoinProject/skywire-mainnet/internal/ioutil	-run TestAckReadWriter  >>./logs/internal/TestAckReadWriter.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/SkycoinProject/skywire-mainnet/internal/ioutil	-run TestAckReadWriterCRCFailure  >>./logs/internal/TestAckReadWriterCRCFailure.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/SkycoinProject/skywire-mainnet/internal/ioutil	-run TestAckReadWriterFlushOnClose  >>./logs/internal/TestAckReadWriterFlushOnClose.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/SkycoinProject/skywire-mainnet/internal/ioutil	-run TestAckReadWriterPartialRead  >>./logs/internal/TestAckReadWriterPartialRead.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/SkycoinProject/skywire-mainnet/internal/ioutil	-run TestAckReadWriterReadError  >>./logs/internal/TestAckReadWriterReadError.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/SkycoinProject/skywire-mainnet/internal/ioutil	-run TestLenReadWriter  >>./logs/internal/TestLenReadWriter.log
```

Notes:

1. `go clean -testcache` - is essential. If ommitted - you will get cached results
2. `-cover` - gives valueable information when test "hangs": coverage will vary on `ok` and `FAILS`
3. `-race` - valueable too
4. `-tags no_ci` - to exclude integration tests

### 2. Collect statistics

Most time-consuming step but it does not require any interaction.

Run produced script(s) multiple times.

E.g. ~20 cycles can be considered "good enough" to find unstable tests.
And ~100 cycles  to ensure instability.

```sh
loop -n 100 "bash ./ci_scripts/run-internal-tests.sh" # use any looping method at your disposal
loop -n 100 "bash ./ci_scripts/run-pkg-tests.sh"
```

### 3. Analyse

```sh
grep "FAIL" ./logs/internal/*.log
grep "FAIL" ./logs/pkg/*.log
```

If you see something like:

```sh
$ grep "FAIL" ./logs/pkg/*.log
# ./logs/pkg/TestClientConnectInitialServers.log:FAIL     github.com/SkycoinProject/skywire-mainnet/pkg/messaging        300.838s
# ./logs/pkg/TestClientConnectInitialServers.log:FAIL     github.com/SkycoinProject/skywire-mainnet/pkg/messaging        300.849s
# ./logs/pkg/TestClientConnectInitialServers.log:FAIL     github.com/SkycoinProject/skywire-mainnet/pkg/messaging        300.844s
# ./logs/pkg/TestClientConnectInitialServers.log:FAIL     github.com/SkycoinProject/skywire-mainnet/pkg/messaging        300.849s
```

(note  300s for FAILs)

And:

```sh
$ grep "coverage" ./logs/pkg/TestClientConnectInitialServers.log  
# ok      github.com/SkycoinProject/skywire-mainnet/pkg/messaging        3.049s  coverage: 39.5% of statements
# coverage: 38.0% of statements
# ok      github.com/SkycoinProject/skywire-mainnet/pkg/messaging        3.072s  coverage: 39.5% of statements
# ok      github.com/SkycoinProject/skywire-mainnet/pkg/messaging        3.073s  coverage: 39.5% of statements
# ok      github.com/SkycoinProject/skywire-mainnet/pkg/messaging        3.071s  coverage: 39.5% of statements
# ok      github.com/SkycoinProject/skywire-mainnet/pkg/messaging        3.050s  coverage: 39.5% of statements
# coverage: 38.0% of statements
```

(note varying coverage)

You have found unstable test.

### 4. Act

Either fix it or tag it with `no_ci` tag.

## History

### 2019-05-15. Commit: 263ba5ce3f8d6b41327beb521d4112906841e257

**Detected unstable test**

It was observed that Travis CI builds randomly fail.

Narrowing search it was found that problem arises in `pkg/messaging` tests.

Using this procedure it was found that problem test is `TestClientConnectInitialServers` in `./pkg/messaging/client_test.go`.

Temporary solution: test was moved to `./pkg/messaging/client_test.go` and tagged with: `!no_ci`

**Stable but possibly incorrect tests**


1. TestReadWriterConcurrentTCP
```sh
$ grep coverage ./logs/internal/*.log
# ./logs/internal/TestReadWriterConcurrentTCP.log
# 1:ok    github.com/SkycoinProject/skywire-mainnet/internal/noise       1.545s  coverage: 0.0% of statements
# 2:ok    github.com/SkycoinProject/skywire-mainnet/internal/noise       1.427s  coverage: 0.0% of statements
# 3:ok    github.com/SkycoinProject/skywire-mainnet/internal/noise       1.429s  coverage: 0.0% of statements
# 4:ok    github.com/SkycoinProject/skywire-mainnet/internal/noise       1.429s  coverage: 0.0% of statements
# 5:ok    github.com/SkycoinProject/skywire-mainnet/internal/noise       1.436s  coverage: 0.0% of statements
```

Note 0.0% coverage

Test removed.
