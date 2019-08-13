// +build !no_ci

package visor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skywire/pkg/routing"
)

func readConfig(cfgFile string) (Config, error) {
	rdr, err := os.Open(filepath.Clean(cfgFile))
	if err != nil {
		return Config{}, err
	}
	conf := Config{}
	if err := json.NewDecoder(rdr).Decode(&conf); err != nil {
		return Config{}, err
	}
	return conf, err
}

func ipPool(addrTemplate string, n uint) []string {
	ipAddrs := make([]string, n)
	for i := uint(1); i < n+1; i++ {
		ipAddrs[i-1] = fmt.Sprintf(addrTemplate, i)
	}
	return ipAddrs
}

func Example_ipPool() {

	ipAddrs := ipPool("12.12.12.%d", 1)
	fmt.Printf("%v\n", ipAddrs)

	// Output: ZZZZ
}

func cfgPool(baseCfg Config, ipTmplt, localPathTmplt string, n uint) []Config {
	nodeCfgs := make([]Config, n)

	for i := uint(0); i < n; i++ {
		ip := fmt.Sprintf(ipTmplt, i+1)
		localPath := fmt.Sprintf(localPathTmplt, i+1)
		pk, sk, _ := cipher.GenerateDeterministicKeyPair([]byte(ip))

		baseCfg.Node.StaticPubKey = pk
		baseCfg.Node.StaticSecKey = sk
		baseCfg.LocalPath = localPath
		baseCfg.Interfaces.RPCAddress = fmt.Sprintf("%s:3435", ip)
		baseCfg.Transport.LogStore.Location = fmt.Sprintf("%s/transport_logs", localPath)
		baseCfg.Routing.Table.Location = fmt.Sprintf("%s/routing.db", localPath)
		baseCfg.Apps = []AppConfig{
			AppConfig{
				App:       "skychat",
				AutoStart: true,
				Port:      routing.Port(1),
				Args: []string{
					"-addr",
					fmt.Sprintf("%v:8001", ip),
				},
			}}
		baseCfg.TCPTransportAddr = fmt.Sprintf("%v:9119", ip)

		nodeCfgs[i] = baseCfg
	}
	return nodeCfgs
}

func Example_cfgPool() {
	cfgFile, err := filepath.Abs("../../integration/tcp-tr/nodeA.json")
	baseCfg, err := readConfig(cfgFile)
	fmt.Printf("readConfig success: %v\n", err == nil)

	nodeCfgs := cfgPool(baseCfg, "12.12.12.%d", "./local/node_%03d", 3)
	fmt.Printf("len(nodeCfgs): %v\n", len(nodeCfgs))

	// Output: readConfig success: true
	// len(nodeCfgs): 3
}

func nodePool(cfgs []Config) []*Node {
	nodes := make([]*Node, len(cfgs))

	var err error
	for i := 0; i < len(cfgs); i++ {
		nodes[i], err = NewNode(&cfgs[i], masterLogger)
		if err != nil {
			panic(err)
		}
	}
	return nodes
}

/* Run in shell:
$ for ((i=1; i<=16; i++)){_ ip addr add 12.12.12.$i/32 dev lo}
*/

func Example_nodePool() {
	cfgFile, err := filepath.Abs("../../integration/tcp-tr/nodeA.json")
	baseCfg, err := readConfig(cfgFile)
	baseCfg.PubKeysFile, _ = filepath.Abs("../../integration/tcp-tr/hosts.pubkeys")
	baseCfg.AppsPath, _ = filepath.Abs("../../apps")

	fmt.Printf("readConfig success: %v\n", err == nil)
	nodeCfgs := cfgPool(baseCfg, "12.12.12.%d", "./local/node_%03d", 16)

	nodes := nodePool(nodeCfgs)
	fmt.Printf("Nodes: 12.12.12.1 - 12.12.12.%d", len(nodes))

	// Output: readConfig success: true
	// Nodes: 12.12.12.1 - 12.12.12.16
}

func startMultiHead(nodes []*Node) chan error {
	errs := make(chan error, len(nodes))
	for i := 0; i < len(nodes); i++ {
		go func(n int) {
			if err := nodes[n].Start(); err != nil {
				errs <- fmt.Errorf("error %v starting node %v", err, n)
			}
		}(i)
	}
	return errs
}

func stopMultiHead(nodes []*Node) chan error {
	errs := make(chan error, len(nodes))
	for i := 0; i < len(nodes); i++ {
		go func(n int) {
			if err := nodes[n].Close(); err != nil {
				errs <- fmt.Errorf("error %v starting node %v", err, n)
			}
		}(i)
	}
	return errs
}

func Example_startMultiHead() {

	_ = os.MkdirAll("/tmp/multihead", 0777)
	f, err := os.OpenFile("/tmp/multihead/multihead.log", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	masterLogger.Out = f

	cfgFile, err := filepath.Abs("../../integration/tcp-tr/nodeA.json")
	baseCfg, err := readConfig(cfgFile)
	baseCfg.PubKeysFile, _ = filepath.Abs("../../integration/tcp-tr/hosts.pubkeys")
	baseCfg.AppsPath, _ = filepath.Abs("../../apps")
	fmt.Printf("baseCfg success: %v\n", err == nil)

	nodeCfgs := cfgPool(baseCfg, "12.12.12.%d", "/tmp/multihead/node_%03d", 128)
	nodes := nodePool(nodeCfgs)

	errsOnStart := startMultiHead(nodes)

	time.Sleep(time.Second * 5)

	errsOnStop := stopMultiHead(nodes)

	time.Sleep(time.Second * 15)

	close(errsOnStart)
	close(errsOnStop)

	fmt.Printf("errsOnStart: %v\n", len(errsOnStart))
	fmt.Printf("errsOnStop: %v\n", len(errsOnStop))

	// Output: ZZZZ
}
