package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/skycoin/skywire/pkg/hypervisor"
	"github.com/skycoin/skywire/pkg/snet"
	"github.com/skycoin/skywire/pkg/util/pathutil"
	"github.com/spf13/cobra"
)

const configEnv = "SW_HYPERVISOR_CONFIG"

var (
	log = logging.MustGetLogger("hypervisor")

	configPath     string
	mock           bool
	mockEnableAuth bool
	mockNodes      int
	mockMaxTps     int
	mockMaxRoutes  int
)

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "./hypervisor-config.json", "hypervisor config path")
	rootCmd.Flags().BoolVarP(&mock, "mock", "m", false, "whether to run hypervisor with mock data")
	rootCmd.Flags().BoolVar(&mockEnableAuth, "mock-enable-auth", false, "whether to enable user management in mock mode")
	rootCmd.Flags().IntVar(&mockNodes, "mock-nodes", 5, "number of app nodes to have in mock mode")
	rootCmd.Flags().IntVar(&mockMaxTps, "mock-max-tps", 10, "max number of transports per mock app node")
	rootCmd.Flags().IntVar(&mockMaxRoutes, "mock-max-routes", 30, "max number of routes per node")
}

var rootCmd = &cobra.Command{
	Use:   "hypervisor",
	Short: "Manages Skywire App Nodes",
	Run: func(_ *cobra.Command, args []string) {
		if configPath == "" {
			configPath = pathutil.FindConfigPath(args, -1, configEnv, pathutil.HypervisorDefaults())
		}

		var config hypervisor.Config
		config.FillDefaults()
		if err := config.Parse(configPath); err != nil {
			log.WithError(err).Fatalln("failed to parse config file")
		}
		fmt.Println(config)

		var (
			httpAddr = config.Interfaces.HTTPAddr
			rpcAddr  = config.Interfaces.RPCAddr
		)

		m, err := hypervisor.NewNode(config)
		if err != nil {
			log.Fatalln("Failed to start hypervisor:", err)
		}

		log.Infof("serving RPC on '%s'", rpcAddr)
		go func() {
			_, rpcPort, err := config.Interfaces.SplitRPCAddr()
			if err != nil {
				log.Fatalln("Failed to parse rpc port from rpc address:", err)
			}

			dmsgConf := snet.Config{
				PubKey:     config.PK,
				SecKey:     config.SK,
				TpNetworks: []string{snet.DmsgType},

				DmsgDiscAddr: config.DmsgDiscovery,
				DmsgMinSrvs:  1,
			}

			network := snet.New(dmsgConf)

			ctx := context.Background()
			if err := network.Init(ctx); err != nil {
				log.Fatalln("failed to init network: %v", err)
			}
			l, err := network.Listen(snet.DmsgType, rpcPort)
			if err != nil {
				log.Fatalln("Failed to bind tcp port:", err)
			}
			if err := m.ServeRPC(l); err != nil {
				log.Fatalln("Failed to serve RPC:", err)
			}
		}()

		if mock {
			err := m.AddMockData(hypervisor.MockConfig{
				Nodes:            mockNodes,
				MaxTpsPerNode:    mockMaxTps,
				MaxRoutesPerNode: mockMaxRoutes,
				EnableAuth:       mockEnableAuth,
			})
			if err != nil {
				log.Fatalln("Failed to add mock data:", err)
			}
		}

		log.Infof("serving HTTP on '%s'", httpAddr)
		if err := http.ListenAndServe(httpAddr, m); err != nil {
			log.Fatalln("Hypervisor exited with error:", err)
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
