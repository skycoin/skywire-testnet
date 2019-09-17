module github.com/SkycoinProject/skywire-mainnet

go 1.12

require (
	github.com/SkycoinProject/dmsg v0.0.0-20190917230949-27f4fd2f25fb
	github.com/SkycoinProject/skycoin v0.26.0
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
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.3.0
	go.etcd.io/bbolt v1.3.3
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/net v0.0.0-20190724013045-ca1201d0de80
	golang.org/x/sys v0.0.0-20190804053845-51ab0e2deafa // indirect
	golang.org/x/tools v0.0.0-20190805222050-c5a2fd39b72a // indirect
)

// Uncomment for tests with alternate branches of 'dmsg'
//replace github.com/SkycoinProject/dmsg => ../dmsg
