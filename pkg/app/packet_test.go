package app

import (
	"fmt"

	"github.com/SkycoinProject/dmsg/cipher"

	"github.com/SkycoinProject/skywire/pkg/routing"
)

func ExamplePacket() {
	var lpk, rpk cipher.PubKey
	laddr := routing.Addr{Port: 0, PubKey: lpk}
	raddr := routing.Addr{Port: 0, PubKey: rpk}
	loop := routing.Loop{Local: laddr, Remote: raddr}

	fmt.Println(raddr.Network())
	fmt.Printf("%v\n", raddr)
	fmt.Printf("%v\n", loop)

	// Output: skywire
	// 000000000000000000000000000000000000000000000000000000000000000000:0
	// 000000000000000000000000000000000000000000000000000000000000000000:0 <-> 000000000000000000000000000000000000000000000000000000000000000000:0
}
