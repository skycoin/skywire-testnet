package nodemanager

type NodeManagerConfig struct {
	Domain           string // domain name of the nodemanager
	CtrlAddr         string // address for talking with nodes
	AppTrackerAddr   string // address of service manager
	RouteManagerAddr string // address of route finding service
	LogisticsServer  string // address of logistics service
}
