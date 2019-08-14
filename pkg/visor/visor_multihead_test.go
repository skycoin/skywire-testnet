// +build !no_ci

package visor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skywire/pkg/routing"
)

type multiheadLog struct {
	mx      sync.Mutex
	records []string
}

func (mhl *multiheadLog) Write(p []byte) (n int, err error) {
	mhl.mx.Lock()
	defer mhl.mx.Unlock()
	mhl.records = append(mhl.records, string(p))
	return len(p), nil
}

/* Prepare IP aliases:
$ for ((i=1; i<=16; i++)){_ ip addr add 12.12.12.$i/32 dev lo}
*/

type MultiHead struct {
	baseCfg    Config
	ipTemplate string
	ipPool     []string
	cfgPool    []Config
	nodes      []*Node

	Log *multiheadLog
}

func initMultiHead() *MultiHead {
	mhLog := multiheadLog{records: []string{}}
	masterLogger.Out = &mhLog
	return &MultiHead{Log: &mhLog}
}

func Example_initMultiHead() {
	mh := initMultiHead()
	mh.Log.Write([]byte("initMultiHead success"))
	fmt.Printf("%v\n", mh.Log.records)

	// Output: [initMultiHead success]
}

func (mh *MultiHead) readBaseConfig(cfgFile string) error {
	rdr, err := os.Open(filepath.Clean(cfgFile))
	if err != nil {
		return err
	}
	baseCfg := Config{}
	if err := json.NewDecoder(rdr).Decode(&baseCfg); err != nil {
		return err
	}
	baseCfg.PubKeysFile, _ = filepath.Abs("../../integration/tcp-tr/hosts.pubkeys")
	baseCfg.AppsPath, _ = filepath.Abs("../../apps")
	baseCfg.Routing.Table.Type = "memory"

	mh.baseCfg = baseCfg
	return nil
}

func Example_readBaseConfig() {
	mh := initMultiHead()
	cfgFile, err := filepath.Abs("../../integration/tcp-tr/nodeA.json")
	err = mh.readBaseConfig(cfgFile)
	fmt.Printf("mh.ReadBaseConfig success: %v\n", err == nil)

	// Output: mh.ReadBaseConfig success: true
}

func (mh *MultiHead) initIPPool(ipTemplate string, n uint) {
	ipPool := make([]string, n)
	for i := uint(1); i < n+1; i++ {
		ipPool[i-1] = fmt.Sprintf(ipTemplate, i)
	}
	mh.ipTemplate = ipTemplate
	mh.ipPool = ipPool

}

func ExampleMultiHead_initIPPool() {
	mh := initMultiHead()
	mh.initIPPool("12.12.12.%d", 16)

	fmt.Printf("IP pool length: %v\n", len(mh.ipPool))

	// Output: IP pool length: 16
}

func (mh *MultiHead) initCfgPool() {
	mh.cfgPool = make([]Config, len(mh.ipPool))
	localPathTmplt := "/tmp/multihead"
	baseCfg := mh.baseCfg

	for i := 0; i < len(mh.ipPool); i++ {
		ip := fmt.Sprintf(mh.ipTemplate, i+1)
		localPath := fmt.Sprintf(localPathTmplt, i+1)
		pk, sk, _ := cipher.GenerateDeterministicKeyPair([]byte(ip))

		baseCfg.Node.StaticPubKey = pk
		baseCfg.Node.StaticSecKey = sk
		baseCfg.LocalPath = localPath
		baseCfg.Interfaces.RPCAddress = fmt.Sprintf("%s:3435", ip)
		baseCfg.Transport.LogStore.Location = fmt.Sprintf("%s/transport_logs", localPath)

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

		mh.cfgPool[i] = baseCfg
	}

}

func Example_initCfgPool() {
	mh := initMultiHead()
	cfgFile, err := filepath.Abs("../../integration/tcp-tr/nodeA.json")
	err = mh.readBaseConfig(cfgFile)
	fmt.Printf("mh.ReadBaseConfig success: %v\n", err == nil)
	mh.initIPPool("12.12.12.%d", 16)
	mh.initCfgPool()

	fmt.Printf("len(mh.cfgPool): %v\n", len(mh.cfgPool))

	// Output: mh.ReadBaseConfig success: true
	// len(mh.cfgPool): 16
}

func (mh *MultiHead) initNodes() chan error {
	mh.nodes = make([]*Node, len(mh.cfgPool))
	errs := make(chan error, len(mh.cfgPool))

	var err error
	for i := 0; i < len(mh.nodes); i++ {
		mh.nodes[i], err = NewNode(&mh.cfgPool[i], masterLogger)
		if err != nil {
			errs <- fmt.Errorf("error %v starting node %v", err, i)
		}
	}
	return errs
}

func ExampleMultiHead_initNodes() {
	mh := initMultiHead()
	cfgFile, err := filepath.Abs("../../integration/tcp-tr/nodeA.json")
	err = mh.readBaseConfig(cfgFile)
	fmt.Printf("mh.ReadBaseConfig success: %v\n", err == nil)
	mh.initIPPool("12.12.12.%d", 16)
	mh.initCfgPool()
	initErrs := mh.initNodes()
	close(initErrs)

	for err := range initErrs {
		fmt.Printf("%v\n", err)
	}
	fmt.Printf("Errors on initNodes: %v\n", len(initErrs))

	// Output: mh.ReadBaseConfig success: true
	// Errors on initNodes: 0
}

func (mh *MultiHead) startNodes() chan error {
	errs := make(chan error, len(mh.nodes))
	for i := 0; i < len(mh.nodes); i++ {
		go func(n int) {
			if err := mh.nodes[n].Start(); err != nil {
				errs <- fmt.Errorf("error %v starting node %v", err, n)
			}
		}(i)
	}
	return errs
}

func (mh *MultiHead) stopNodes() chan error {
	errs := make(chan error, len(mh.nodes))
	for i := 0; i < len(mh.nodes); i++ {
		go func(n int) {
			if err := mh.nodes[n].Close(); err != nil {
				errs <- fmt.Errorf("error %v starting node %v", err, n)
			}
		}(i)
	}
	return errs
}

func makeMultiHeadN(n uint) *MultiHead {
	mh := initMultiHead()
	cfgFile, err := filepath.Abs("../../integration/tcp-tr/nodeA.json")
	err = mh.readBaseConfig(cfgFile)
	fmt.Printf("mh.ReadBaseConfig success: %v\n", err == nil)
	mh.initIPPool("12.12.12.%d", n)
	mh.initCfgPool()

	return mh
}

func ExampleMultiHead_startNodes() {
	mh := initMultiHead()
	cfgFile, err := filepath.Abs("../../integration/tcp-tr/nodeA.json")
	err = mh.readBaseConfig(cfgFile)
	fmt.Printf("mh.ReadBaseConfig success: %v\n", err == nil)
	mh.initIPPool("12.12.12.%d", 16)
	mh.initCfgPool()
	initErrs := mh.initNodes()
	close(initErrs)

	startErrs := mh.startNodes()
	time.Sleep(time.Second * 5)
	stopErrs := mh.stopNodes()
	time.Sleep(time.Second * 5)

	close(startErrs)
	close(stopErrs)

	for err := range initErrs {
		fmt.Printf("%v\n", err)
	}
	fmt.Printf("Errors on initNodes: %v\n", len(initErrs))

	for err := range startErrs {
		fmt.Printf("%v\n", err)
	}
	fmt.Printf("Errors on startNodes: %v\n", len(startErrs))

	for err := range stopErrs {
		fmt.Printf("%v\n", err)
	}
	fmt.Printf("Errors on stopNodes: %v\n", len(stopErrs))

	// Output: mh.ReadBaseConfig success: true
	// Errors on initNodes: 0
	// Errors on startNodes: 0
	// Errors on stopNodes: 0
}

func (mh *MultiHead) sendMessage(sender, reciever uint, message string) (*http.Response, error) {
	url := fmt.Sprintf("http://%s:8001/message", mh.ipPool[sender])
	msgData := map[string]string{
		"recipient": fmt.Sprintf("%s", mh.cfgPool[reciever].Node.StaticPubKey),
		"message":   "Hello",
	}
	data, _ := json.Marshal(msgData)                                 // nolint: errcheck
	return http.Post(url, "application/json", bytes.NewBuffer(data)) // nolint: gosec

}

func ExampleMultiHead_sendMessage() {
	mh := makeMultiHeadN(2)

	initErrs := mh.initNodes()
	startErrs := mh.startNodes()
	time.Sleep(time.Second * 2)

	// fmt.Printf("resp: %v, err: %v\n", resp, err)

	resp, err := mh.sendMessage(0, 1, "Hello")
	fmt.Printf("resp: %v, err: %v\n ", resp, err)

	stopErrs := mh.stopNodes()
	time.Sleep(time.Second * 5)

	close(initErrs)
	close(startErrs)
	close(stopErrs)

	for err := range initErrs {
		fmt.Printf("%v\n", err)
	}
	fmt.Printf("Errors on initNodes: %v\n", len(initErrs))

	for err := range startErrs {
		fmt.Printf("%v\n", err)
	}
	fmt.Printf("Errors on startNodes: %v\n", len(startErrs))

	for err := range stopErrs {
		fmt.Printf("%v\n", err)
	}
	fmt.Printf("Errors on stopNodes: %v\n", len(stopErrs))

	fmt.Printf("%v\n", mh.Log.records)

	// Output: ZZZ
}
