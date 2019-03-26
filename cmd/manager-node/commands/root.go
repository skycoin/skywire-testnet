package commands

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/manager"
)

var (
	pk            cipher.PubKey
	sk            cipher.SecKey
	rpcAddr       string
	httpAddr      string
	mock          bool
	mockNodes     int
	mockMaxTps    int
	mockMaxRoutes int

	log = logging.MustGetLogger("manager-node")
)

func init() {
	rootCmd.PersistentFlags().Var(&pk, "pk", "manager node's public key")
	rootCmd.PersistentFlags().Var(&sk, "sk", "manager node's secret key")
	rootCmd.PersistentFlags().StringVar(&httpAddr, "http-addr", ":8080", "address to serve HTTP RESTful API and Web interface")

	rootCmd.Flags().StringVar(&rpcAddr, "rpc-addr", ":7080", "address to serve RPC client interface")
	rootCmd.Flags().BoolVar(&mock, "mock", false, "whether to run manager node with mock data")
	rootCmd.Flags().IntVar(&mockNodes, "mock-nodes", 5, "number of app nodes to have in mock mode")
	rootCmd.Flags().IntVar(&mockMaxTps, "mock-max-tps", 10, "max number of transports per mock app node")
	rootCmd.Flags().IntVar(&mockMaxRoutes, "mock-max-routes", 10, "max number of routes per node")
}

var rootCmd = &cobra.Command{
	Use:   "manager-node",
	Short: "Manages Skywire App Nodes",
	PreRun: func(_ *cobra.Command, _ []string) {
		if pk.Null() && sk.Null() {
			pk, sk = cipher.GenerateKeyPair()
			log.Println("No keys are set. Randomly generating...")
		}
		cPK, err := sk.PubKey()
		if err != nil {
			log.Fatalln("Key pair check failed:", err)
		}
		if cPK != pk {
			log.Fatalln("SK and PK provided do not match.")
		}
		log.Println("PK:", pk)
		log.Println("SK:", sk)
	},
	Run: func(_ *cobra.Command, _ []string) {
		m, err := manager.NewNode(manager.MakeConfig("manager.db")) // TODO: complete
		if err != nil {
			log.Fatalln("Failed to start manager:", err)
		}
		log.Infof("serving  RPC on '%s'", rpcAddr)
		go func() {
			l, err := net.Listen("tcp", rpcAddr)
			if err != nil {
				log.Fatalln("Failed to bind tcp port:", err)
			}
			if err := m.ServeRPC(l); err != nil {
				log.Fatalln("Failed to serve RPC:", err)
			}
		}()
		if mock {
			err := m.AddMockData(&manager.MockConfig{
				Nodes:            mockNodes,
				MaxTpsPerNode:    mockMaxTps,
				MaxRoutesPerNode: mockMaxRoutes,
			})
			if err != nil {
				log.Fatalln("Failed to add mock data:", err)
			}
		}
		log.Infof("serving HTTP on '%s'", httpAddr)
		if err := http.ListenAndServe(httpAddr, m); err != nil {
			log.Fatalln("Manager exited with error:", err)
		}
		log.Println("Good bye!")
	},
}

// Execute executes root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
