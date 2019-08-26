// +build multihead, !no_ci

package visor

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/skycoin/skywire/internal/testhelpers"
)

func Example_newMultiHead() {

	mh := newMultiHead()
	_, _ = mh.Log.Write([]byte("initMultiHead success")) //nolint: errcheck
	fmt.Printf("%v\n", mh.Log.records)

	// Output: [initMultiHead success]
}

func Example_readBaseConfig() {
	mh := newMultiHead()
	cfgFile, _ := filepath.Abs("testdata/node.json") //nolint: errcheck
	err := mh.readBaseConfig(cfgFile)
	fmt.Printf("mh.ReadBaseConfig success: %v\n", err == nil)

	// Output: mh.ReadBaseConfig success: true
}

func ExampleMultiHead_initIPPool() {
	mh := newMultiHead()
	mh.initIPPool("12.12.12.%d", 16)
	fmt.Printf("IP pool length: %v\n", len(mh.ipPool))

	// Output: IP pool length: 16
}

func Example_initCfgPool() {
	mh := newMultiHead()
	cfgFile, _ := filepath.Abs("../../integration/tcp-tr/nodeA.json") //nolint: errcheck
	err := mh.readBaseConfig(cfgFile)
	fmt.Printf("mh.ReadBaseConfig success: %v\n", err == nil)
	mh.initIPPool("skyhost%03d", 16)
	mh.initCfgPool()
	fmt.Printf("len(mh.cfgPool): %v\n", len(mh.cfgPool))

	// Output: mh.ReadBaseConfig success: true
	// len(mh.cfgPool): 16
}

func ExampleMultiHead_initNodes() {
	mh := newMultiHead()
	cfgFile, _ := filepath.Abs("testdata/node.json") //nolint: errcheck
	err := mh.readBaseConfig(cfgFile)
	fmt.Printf("mh.ReadBaseConfig success: %v\n", err == nil)
	mh.initIPPool("skyhost_%03d", 1)
	mh.initCfgPool()
	mh.initNodes()
	fmt.Print(mh.errReport())

	// Output: mh.ReadBaseConfig success: true
	// init errors: 0
	// start errors: 0
	// stop errors: 0
}

func TestMultiHead_startNodes(t *testing.T) {
	delay := time.Second
	mh := makeMultiHeadN(32)
	mh.startNodes(delay)
	mh.stopNodes(delay)

	testhelpers.NoErrorWithinTimeoutN(
		mh.initErrs, mh.startErrs, mh.stopErrs,
	)
}

func TestMultiHead_genPubKeysFile(t *testing.T) {
	mh := makeMultiHeadN(128)
	pkFile := mh.genPubKeysFile()

	fmt.Println(pkFile)
	t.Log(pkFile)
}
