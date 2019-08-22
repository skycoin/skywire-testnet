module github.com/skycoin/skywire

go 1.12

require (
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/creack/pty v1.1.7
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/google/uuid v1.1.1
	github.com/gorilla/handlers v1.4.0
	github.com/gorilla/securecookie v1.1.1
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d
	github.com/kr/pty v1.1.5 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/profile v1.3.0
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/common v0.4.1
	github.com/sirupsen/logrus v1.4.2
	github.com/skycoin/dmsg v0.0.0-20190816104216-d18ee6aa05cb
	github.com/skycoin/skycoin v0.26.0
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.3.0
	go.etcd.io/bbolt v1.3.3
	golang.org/x/crypto v0.0.0-20190820162420-60c769a6c586
	golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7
	golang.org/x/sys v0.0.0-20190813064441-fde4db37ae7a // indirect
	golang.org/x/tools v0.0.0-20190821162956-65e3620a7ae7 // indirect
)

// Uncomment for tests with alternate branches of 'dmsg'
replace github.com/skycoin/dmsg => ../dmsg
