package commands

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/internal/pathutil"
	"github.com/skycoin/skywire/pkg/manager"
)

const configEnv = "SW_MANAGER_CONFIG"

var (
	log = logging.MustGetLogger("manager-node")

	mock          bool
	mockNodes     int
	mockMaxTps    int
	mockMaxRoutes int
)

func init() {
	rootCmd.Flags().BoolVar(&mock, "mock", false, "whether to run manager node with mock data")
	rootCmd.Flags().IntVar(&mockNodes, "mock-nodes", 5, "number of app nodes to have in mock mode")
	rootCmd.Flags().IntVar(&mockMaxTps, "mock-max-tps", 10, "max number of transports per mock app node")
	rootCmd.Flags().IntVar(&mockMaxRoutes, "mock-max-routes", 10, "max number of routes per node")
}

var rootCmd = &cobra.Command{
	Use:   "manager-node [config-path]",
	Short: "Manages Skywire App Nodes",
	Run: func(_ *cobra.Command, args []string) {
		configPath := pathutil.FindConfigPath(args, 0, configEnv, pathutil.ManagerDefaults())

		var config manager.Config
		config.FillDefaults()
		if err := config.Parse(configPath); err != nil {
			log.WithError(err).Fatalln("failed to parse config file")
		}

		var (
			httpAddr = config.Interfaces.HTTPAddr
			rpcAddr  = config.Interfaces.RPCAddr
		)

		m, err := manager.NewNode(config)
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
			err := m.AddMockData(manager.MockConfig{
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
