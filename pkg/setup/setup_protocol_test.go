package setup

import (
	"fmt"
	"net"

	"github.com/skycoin/skywire/pkg/routing"
)

func ExampleNewSetupProtocol() {
	in, _ := net.Pipe()
	defer in.Close()

	sProto := NewSetupProtocol(in)
	fmt.Printf("Success: %v\n", sProto != nil)

	// Output: Success: true
}

func Example_sendCmd() {
	in, out := net.Pipe()
	defer in.Close()
	defer out.Close()
	inProto, outProto := NewSetupProtocol(in), NewSetupProtocol(out)

	fmt.Printf("Success: %v\n", inProto != nil)

	errCh := make(chan error)
	go func(sProto *Protocol) {
		frame, err := sProto.readFrame()
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Printf("packet: %v, payload: %v\n", Packet(frame[0]), string(frame[1:]))
		errCh <- err
	}(inProto)

	if err := outProto.sendCMD(PacketCreateLoop, &routing.Loop{}); err != nil {
		fmt.Println(err.Error())
	}

	if err := <-errCh; err != nil {
		fmt.Println(err.Error())
	}

	// Output: Success: true
	// packet: CreateLoop, payload: {"LocalPort":0,"RemotePort":0,"Forward":null,"Reverse":null,"ExpireAt":"0001-01-01T00:00:00Z","NoiseMessage":null}

}

func ExampleProtocol_ReadPacket() {
	in, out := net.Pipe()
	defer in.Close()
	defer out.Close()
	inProto, outProto := NewSetupProtocol(in), NewSetupProtocol(out)

	fmt.Printf("Success: %v\n", inProto != nil)

	errCh := make(chan error)
	go func(sProto *Protocol) {
		packet, payload, err := sProto.ReadPacket()
		if err != nil {
			fmt.Println(err.Error())
		}

		fmt.Printf("packet: %v, payload: %v\n", packet, string(payload))
		errCh <- err
	}(inProto)

	if err := outProto.sendCMD(PacketCreateLoop, &routing.Loop{}); err != nil {
		fmt.Println(err.Error())
	}

	if err := <-errCh; err != nil {
		fmt.Println(err.Error())
	}

	// Output: Success: true
	// packet: CreateLoop, payload: {"LocalPort":0,"RemotePort":0,"Forward":null,"Reverse":null,"ExpireAt":"0001-01-01T00:00:00Z","NoiseMessage":null}
}

func ExampleProtocol_CreateLoop() {
	in, out := net.Pipe()
	defer in.Close()
	defer out.Close()
	inProto, outProto := NewSetupProtocol(in), NewSetupProtocol(out)
	fmt.Printf("Success: %v\n", inProto != nil)

	// errCh := make(chan error)
	go func(sProto *Protocol) {
		packet, payload, err := sProto.ReadPacket()
		if err != nil {
			fmt.Println(err.Error())
		}

		fmt.Printf("packet: %v, payload: %v\n", packet, string(payload))

		inProto.Respond(nil)
	}(inProto)

	// TODO: create non-empty loop
	loop := &routing.Loop{}
	if err := outProto.CreateLoop(loop); err != nil {
		fmt.Println(err.Error())
	}

	// Output: Success: true
	// packet: CreateLoop, payload: {"LocalPort":0,"RemotePort":0,"Forward":null,"Reverse":null,"ExpireAt":"0001-01-01T00:00:00Z","NoiseMessage":null}
}

/*
func ExampleProtocol_ConfirmLoop() {
	fmt.Println("TODO")
	// Output: Success
}

func ExampleProtocol_CloseLoop() {
	fmt.Println("TODO")
	// Output: Success
}

func ExampleProtocol_LoopClosed() {
	fmt.Println("TODO")
	// Output: Success
}
*/
