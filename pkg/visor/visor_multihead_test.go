// +build !no_ci

package visor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	Log       *multiheadLog
	initErrs  chan error
	startErrs chan error
	stopErrs  chan error
	sync.Once
}

func initMultiHead() *MultiHead {
	mhLog := multiheadLog{records: []string{}}
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

func (mh *MultiHead) initNodes() {
	mh.nodes = make([]*Node, len(mh.cfgPool))
	mh.initErrs = make(chan error, len(mh.cfgPool))

	subs := []struct{ old, new string }{
		{"000000000000000000000000000000000000000000000000000000000000000000", "PubKey{}"},
		{"024195ae0d46eb0195c9ddabcaf62bc894316594ea2e92570f269238c5b5f817d1", "PK(skyhost_001)"},
	}

	var err error
	for i := 0; i < len(mh.nodes); i++ {
		logger := NewTaggedMasterLogger(fmt.Sprintf("[node_%03d]", i+1), subs)
		logger.Out = mh.Log
		mh.nodes[i], err = NewNode(&mh.cfgPool[i], logger)
		if err != nil {
			mh.initErrs <- fmt.Errorf("error %v starting node %v", err, i)
		}
	}
}

func ExampleMultiHead_initNodes() {
	mh := initMultiHead()
	cfgFile, err := filepath.Abs("../../integration/tcp-tr/nodeA.json")
	err = mh.readBaseConfig(cfgFile)
	fmt.Printf("mh.ReadBaseConfig success: %v\n", err == nil)
	mh.initIPPool("skyhost_%03d", 16)
	mh.initCfgPool()
	mh.initNodes()

	close(mh.initErrs)
	for err := range mh.initErrs {
		fmt.Printf("%v\n", err)
	}
	fmt.Printf("Errors on initNodes: %v\n", len(mh.initErrs))

	// Output: mh.ReadBaseConfig success: true
	// Errors on initNodes: 0
}

func (mh *MultiHead) startNodes(postdelay time.Duration) {
	mh.startErrs = make(chan error, len(mh.nodes))
	for i := 0; i < len(mh.nodes); i++ {
		go func(n int) {
			if err := mh.nodes[n].Start(); err != nil {
				mh.startErrs <- fmt.Errorf("error %v starting node %v", err, n)
			}
		}(i)
	}
	time.Sleep(postdelay)
}

func (mh *MultiHead) stopNodes(predelay time.Duration) {
	time.Sleep(predelay)
	mh.stopErrs = make(chan error, len(mh.nodes))
	for i := 0; i < len(mh.nodes); i++ {
		go func(n int) {
			if err := mh.nodes[n].Close(); err != nil {
				mh.stopErrs <- fmt.Errorf("error %v starting node %v", err, n)
			}
		}(i)
	}
}

func makeMultiHeadN(n uint) *MultiHead {
	mh := initMultiHead()
	cfgFile, err := filepath.Abs("../../integration/tcp-tr/nodeA.json")
	err = mh.readBaseConfig(cfgFile)
	if err != nil {
		panic(err)
	}
	mh.initIPPool("skyhost_%03d", n)
	mh.initCfgPool()
	mh.initNodes()
	return mh
}

func (mh *MultiHead) errReport() string {
	mh.Do(func() {
		close(mh.initErrs)
		close(mh.startErrs)
		close(mh.stopErrs)
	})
	collectErrs := func(tmplt string, errs chan error) string {
		recs := make([]string, len(errs)+1)
		for err := range errs {
			recs = append(recs, fmt.Sprintf("%v", err))
		}
		recs = append(recs, fmt.Sprintf(tmplt, len(errs)))
		return strings.Join(recs, "\n")
	}
	return strings.Join([]string{
		collectErrs("init errors: %v", mh.initErrs),
		collectErrs("start errors: %v", mh.startErrs),
		collectErrs("stop errors: %v", mh.stopErrs),
	}, "")
}

func ExampleMultiHead_startNodes() {
	delay := time.Second
	n := uint(32)
	mh := makeMultiHeadN(n)
	mh.startNodes(delay)

	mh.stopNodes(delay)
	fmt.Printf("%v\n", mh.errReport())
	fmt.Printf("records: %v,  %v\n", len(mh.Log.records), len(mh.Log.records) >= int(n*12))
	// Output: init errors: 0
	// start errors: 0
	// stop errors: 0
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

// WIP
func ExampleMultiHead_sendMessage() {
	mh := makeMultiHeadN(2)
	mh.startNodes(time.Second)

	_, err := mh.sendMessage(0, 0, "Hello")
	fmt.Printf("err: %v", err)

	mh.stopNodes(time.Second * 3)
	fmt.Printf("%v\n", mh.errReport())
	fmt.Printf("%v\n", mh.Log.records)

	// Output: err: <nil>
	// init errors: 0
	// start errors: 0
	// stop errors: 0

}
