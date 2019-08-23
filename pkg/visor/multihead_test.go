// +build !no_ci

package visor

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/router"
	"github.com/skycoin/skywire/pkg/routing"
)

/* Prepare IP aliases:
$ for ((i=1; i<=16; i++)){sudo ip addr add 12.12.12.$i/32 dev lo}
*/

func ExampleMultiHead_RunExample() {
	mh := makeMultiHeadN(3)

	mh.RunExample(func(mh *MultiHead) {
		fmt.Print("Hi! Testing start-stop")
	})
	fmt.Printf("%v\n", mh.errReport())

	// Output:
	// Hi! Testing start-stop
	// init errors: 0
	// start errors: 0
	// stop errors: 0
}

func ExampleMultiHead_RunExample_logrus() {
	mh := makeMultiHeadN(3)

	mh.RunExample(func(mh *MultiHead) {
		fmt.Print("Hi! Testing start-stop")
	})
	fmt.Printf("%v\n", mh.errReport())

	// Output:
	// Hi! Testing start-stop
	// init errors: 0
	// start errors: 0
	// stop errors: 0
}

func ExampleMultiHead_sendMessage_Local() {
	mh := makeMultiHeadN(2)
	mh.RunExample(func(mh *MultiHead) {
		resp, err := mh.sendMessage(0, 0, "Hello001")
		fmt.Printf("resp.Status: %v err: %v", resp.StatusCode, err)
		resp2, err2 := mh.sendMessage(1, 1, "Hello002")
		fmt.Printf("resp.Status: %v err: %v", resp2.StatusCode, err2)
	})

	fmt.Printf("%v\n", mh.errReport())

	recs := mh.Log.filter(`.*received.*"message"`, true)
	if len(recs) == 2 {
		fmt.Println("received message")
	}
	fmt.Println(mh.Log.records)
	fmt.Println(recs)

	// Output: err: <nil>
	// init errors: 0
	// start errors: 0
	// stop errors: 0
	// skyhost_001 received message
}

func TestMultiHead_sendMessage_Local(t *testing.T) {
	mh := makeMultiHeadN(1)
	mh.RunTest(t, "Local", func(mh *MultiHead) {
		resp, err := mh.sendMessage(1, 1, "Hello001")
		require.NoError(t, err)
		require.Equal(t, resp.StatusCode, 200)
	})

	fmt.Printf("%v\n", mh.errReport())

	recs := mh.Log.filter(`skyhost_001.*received.*"message"`, true)
	if len(recs) == 1 {
		fmt.Println("skyhost_001 received message")
	}

	// Output: err: <nil>
	// init errors: 0
	// start errors: 0
	// stop errors: 0
	// skyhost_001 received message
}

func ExampleMultiHead_sendMessage_msg255() {
	mh := makeMultiHeadN(2)
	mh.RunExample(func(mh *MultiHead) {
		for i := uint(2); i < 120; i++ {
			resp, err := mh.sendMessage(i, i+1, fmt.Sprintf("Hello%03d-%03d", i, i+1))
			fmt.Printf("resp: %v \nerr: %v\n", resp, err)
			resp2, err2 := mh.sendMessage(i+1, i, fmt.Sprintf("Hello%03d-%03d", i+1, i))
			fmt.Printf("resp: %v \nerr: %v\n", resp2, err2)
			time.Sleep(time.Millisecond * 10)
		}
		time.Sleep(time.Second * 60)
	})

	fmt.Printf("%v\n", strings.Join(mh.Log.records, ""))
	fmt.Println(mh.Log.filter(`.*received.*`, true))

	// Output: err: <nil>
	// init errors: 0
	// start errors: 0
	// stop errors: 0

}

func ExampleMultiHead_sendMessage_TCP() {
	mh := makeMultiHeadN(3)
	mh.RunExample(func(mh *MultiHead) {
		resp, err := mh.sendMessage(1, 2, "Hello")
		fmt.Printf("resp: %v \nerr: %v\n", resp, err)
		time.Sleep(time.Second)
	})

	fmt.Printf("%v\n", strings.Join(mh.Log.records, ""))
	fmt.Println(mh.Log.filter(`.*received.*`, true))

	// Output: err: <nil>
	// init errors: 0
	// start errors: 0
	// stop errors: 0

}

func ExampleMultihead_forwardAppPacket() {
	mh := makeMultiHeadN(2)
	mh.RunExample(func(mh *MultiHead) {
		cfgA, _ := mh.cfgPool[0], mh.cfgPool[1]
		nodeA, nodeB := mh.nodes[0], mh.nodes[1]
		pkA, _ := nodeA.config.Node.StaticPubKey, nodeB.config.Node.StaticPubKey
		tmA, _ := nodeA.tm, nodeB.tm

		rA, ok := nodeA.router.(*router.Router)
		fmt.Printf("rA is %T %v\n", rA, ok)
		rB, ok := nodeB.router.(*router.Router)
		fmt.Printf("rB is %T %v\n", rB, ok)

		rtA, err := cfgA.RoutingTable()
		fmt.Printf("rtA is %T %v\n", rtA, err)
		rtB, err := cfgA.RoutingTable()
		fmt.Printf("rtB is %T %v\n", rtB, err)

		trA, err := tmA.CreateDataTransport(context.TODO(), pkA, "tcp-transport", true)
		fmt.Printf("CreateDataTransport trA %T success: %v\n", trA, err == nil)

		ruleA := routing.ForwardRule(time.Now().Add(time.Hour), 4, trA.Entry.ID)
		fmt.Printf("ruleA: %v\n", ruleA)

		routeA, err := rtA.AddRule(ruleA)
		fmt.Printf("routeA: %v, err: %v\n", routeA, err)

		// time.Sleep(100 * time.Millisecond)
		packetA := routing.MakePacket(routeA, []byte("Hello"))
		fmt.Printf("packetA: %v\n", packetA)

		n, err := trA.Write(packetA)
		fmt.Printf("trA.Write(packetA): %v %v\n", n, err)
	})

	_, _ = os.Stderr.WriteString(fmt.Sprintf("%v\n", mh.errReport()))                   //nolint: errcheck
	_, _ = os.Stderr.WriteString(fmt.Sprintf("%v\n", strings.Join(mh.Log.records, ""))) //nolint: errcheck
	// fmt.Printf("%v\n", mh.Log.filter(`skychat`, false))

	// Output: ZZZZ
}

func ExampleMultiHead_sendMessage_msg003() {
	mh := makeMultiHeadN(128)
	mh.RunExample(func(mh *MultiHead) {
		_, err := mh.sendMessage(0, 0, "Hello003")
		fmt.Printf("err: %v", err)

	})
	fmt.Printf("%v\n", mh.errReport())
	fmt.Printf("%v\n", strings.Join(mh.Log.records, ""))

	// Output: err: <nil>
	// init errors: 0
	// start errors: 0
	// stop errors: 0
}
