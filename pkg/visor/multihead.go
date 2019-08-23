//+build !no_ci

package visor

import (
	"bytes"
	"encoding/json"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

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

func (mhl *multiheadLog) filter(pattern string, match bool) []string {
	var result []string
	r := regexp.MustCompile(pattern)
	for _, line := range mhl.records {
		matched := r.MatchString(line)
		if (matched && match) || (!matched && !match) {
			result = append(result, line)
		}
	}
	return result
}

// MultiHead allows to run group of nodes in the single process
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

	repOnce   sync.Once
	startOnce sync.Once
	stopOnce  sync.Once
}

func newMultiHead() *MultiHead {
	mhLog := multiheadLog{records: []string{}}
	return &MultiHead{Log: &mhLog}
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
	baseCfg.PubKeysFile, _ = filepath.Abs("testdata/hosts.pubkeys") // nolint: errcheck
	baseCfg.AppsPath, _ = filepath.Abs("../../apps")                // nolint: errcheck
	baseCfg.Routing.Table.Type = "memory"

	mh.baseCfg = baseCfg
	return nil
}

func (mh *MultiHead) initIPPool(ipTemplate string, n uint) {

	ipPool := make([]string, n)
	for i := uint(1); i < n+1; i++ {
		ipPool[i-1] = fmt.Sprintf(ipTemplate, i)
	}
	mh.ipTemplate = ipTemplate
	mh.ipPool = ipPool

}

func (mh *MultiHead) initCfgPool() {
	mh.cfgPool = make([]Config, len(mh.ipPool))
	localPathTmplt := "/tmp/multihead"
	baseCfg := mh.baseCfg

	for i := 0; i < len(mh.ipPool); i++ {
		ip := fmt.Sprintf(mh.ipTemplate, i+1)
		localPath := fmt.Sprintf(localPathTmplt, i+1)
		pk, sk, _ := cipher.GenerateDeterministicKeyPair([]byte(ip)) //nolint: errcheck

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

func (mh *MultiHead) pkAliases() []strsub {
	subs := make([]strsub, len(mh.cfgPool)+1)
	for i := 0; i < len(mh.cfgPool); i++ {
		subs[i] = strsub{
			old: fmt.Sprintf("%v", mh.cfgPool[i].Node.StaticPubKey),
			new: fmt.Sprintf("PK(%v)", mh.ipPool[i]),
		}
	}
	return append(subs, strsub{
		fmt.Sprintf("%v", cipher.PubKey{}), "PK{NULL}"})
}

func (mh *MultiHead) initNodes() {
	mh.nodes = make([]*Node, len(mh.cfgPool))
	mh.initErrs = make(chan error, len(mh.cfgPool))
	mh.startErrs = make(chan error, len(mh.cfgPool))
	mh.stopErrs = make(chan error, len(mh.cfgPool))

	logging.DisableColors()
	logging.SetOutputTo(mh.Log)
	stdlog.SetOutput(mh.Log)
	stdlog.SetPrefix("[package-level]")

	var err error
	for i := 0; i < len(mh.nodes); i++ {
		logger := NewTaggedMasterLogger(mh.ipPool[i], mh.pkAliases())
		logger.Out = mh.Log

		mh.nodes[i], err = NewNode(&mh.cfgPool[i], logger)
		if err != nil {
			mh.initErrs <- fmt.Errorf("error %v starting node %v", err, i)
		}
	}
}

func (mh *MultiHead) startNodes(postdelay time.Duration) {
	mh.startOnce.Do(func() {
		mh.startErrs = make(chan error, len(mh.nodes))
		for i := 0; i < len(mh.nodes); i++ {
			go func(n int) {
				if err := mh.nodes[n].Start(); err != nil {
					mh.startErrs <- fmt.Errorf("error %v starting node %v", err, n)
				}
			}(i)
		}
		time.Sleep(postdelay)
	})
}

func (mh *MultiHead) stopNodes(predelay time.Duration) {
	mh.stopOnce.Do(func() {
		time.Sleep(predelay)
		mh.stopErrs = make(chan error, len(mh.nodes))
		for i := 0; i < len(mh.nodes); i++ {
			go func(n int) {
				if err := mh.nodes[n].Close(); err != nil {
					mh.stopErrs <- fmt.Errorf("error %v stopping node %v", err, n)
				}
			}(i)
		}
	})
}

func makeMultiHeadN(n uint) *MultiHead {
	mh := newMultiHead()
	cfgFile, _ := filepath.Abs("../../integration/tcp-tr/nodeA.json") //nolint: errcheck
	err := mh.readBaseConfig(cfgFile)
	if err != nil {
		panic(err)
	}
	mh.initIPPool("skyhost_%03d", n)
	mh.initCfgPool()
	mh.initNodes()
	return mh
}

func (mh *MultiHead) errReport() string {
	mh.repOnce.Do(func() {
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

func (mh *MultiHead) genPubKeysFile() string {
	recs := make([]string, len(mh.cfgPool))
	for i := 0; i < len(mh.cfgPool); i++ {
		recs[i] = fmt.Sprintf("%s\t%s\n", mh.cfgPool[i].Node.StaticPubKey, mh.ipPool[i])
	}
	return strings.Join(recs, "")
}

func (mh *MultiHead) sendMessage(sender, receiver uint, message string) (*http.Response, error) {
	url := fmt.Sprintf("http://%s:8001/message", mh.ipPool[sender])
	msgData := map[string]string{
		"recipient": mh.cfgPool[receiver].Node.StaticPubKey.String(),
		"message":   message,
	}
	data, _ := json.Marshal(msgData)                                 // nolint: errcheck
	return http.Post(url, "application/json", bytes.NewBuffer(data)) // nolint: gosec

}

// RunExample  runs Example-function in multihead env
func (mh *MultiHead) RunExample(example func(mh *MultiHead)) {
	mh.startNodes(time.Second)
	example(mh)
	mh.stopNodes(time.Second)
}

// RunTest  runs test function in multihead env
func (mh *MultiHead) RunTest(t *testing.T, name string, tfunc func(mh *MultiHead)) {
	mh.startNodes(time.Second)
	t.Run(name, func(t *testing.T) {
		tfunc(mh)
	})
	mh.stopNodes(time.Second)
}
