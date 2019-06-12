# commit a70894c8c4223424151cdff7441b1fb2e6bad309
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/httpauth -run TestClient >> ./logs/internal/TestClient.log

go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/ioutil	-run TestAckReadWriter >> ./logs/internal/TestAckReadWriter.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/ioutil	-run TestAckReadWriterCRCFailure >> ./logs/internal/TestAckReadWriterCRCFailure.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/ioutil	-run TestAckReadWriterFlushOnClose >> ./logs/internal/TestAckReadWriterFlushOnClose.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/ioutil	-run TestAckReadWriterPartialRead >> ./logs/internal/TestAckReadWriterPartialRead.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/ioutil	-run TestAckReadWriterReadError >> ./logs/internal/TestAckReadWriterReadError.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/ioutil	-run TestLenReadWriter >> ./logs/internal/TestLenReadWriter.log

go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/noise -run TestRPCClientDialer >> ./logs/internal/TestRPCClientDialer.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/noise -run TestConn >> ./logs/internal/TestConn.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/noise -run TestListener >> ./logs/internal/TestListener.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/noise -run TestKKAndSecp256k1 >> ./logs/internal/TestKKAndSecp256k1.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/noise -run TestXKAndSecp256k1 >> ./logs/internal/TestXKAndSecp256k1.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/noise -run TestReadWriterKKPattern >> ./logs/internal/TestReadWriterKKPattern.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/noise -run TestReadWriterXKPattern >> ./logs/internal/TestReadWriterXKPattern.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/noise -run TestReadWriterConcurrentTCP >> ./logs/internal/TestReadWriterConcurrentTCP.log

go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/internal/therealproxy -run TestProxy >> ./logs/internal/TestProxy.log

go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/skyssh/internal/therealssh  -run TestListAuthoriser >> ./logs/internal/TestListAuthoriser.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/skyssh/internal/therealssh  -run TestFileAuthoriser >> ./logs/internal/TestFileAuthoriser.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/skyssh/internal/therealssh  -run TestChannelServe >> ./logs/internal/TestChannelServe.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/skyssh/internal/therealssh  -run TestChannelSendWrite >> ./logs/internal/TestChannelSendWrite.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/skyssh/internal/therealssh  -run TestChannelRead >> ./logs/internal/TestChannelRead.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/skyssh/internal/therealssh  -run TestChannelRequest >> ./logs/internal/TestChannelRequest.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/skyssh/internal/therealssh  -run TestChannelServeSocket >> ./logs/internal/TestChannelServeSocket.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/skyssh/internal/therealssh  -run TestClientOpenChannel >> ./logs/internal/TestClientOpenChannel.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/skyssh/internal/therealssh  -run TestClientHandleResponse >> ./logs/internal/TestClientHandleResponse.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/skyssh/internal/therealssh  -run TestClientHandleData >> ./logs/internal/TestClientHandleData.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/skyssh/internal/therealssh  -run TestServerOpenChannel >> ./logs/internal/TestServerOpenChannel.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/skyssh/internal/therealssh  -run TestServerHandleRequest >> ./logs/internal/TestServerHandleRequest.log
go clean -testcache &>/dev/null || go test -race -tags no_ci -cover -timeout=5m github.com/skycoin/skywire/skyssh/internal/therealssh  -run TestServerHandleData >> ./logs/internal/TestServerHandleData.log
