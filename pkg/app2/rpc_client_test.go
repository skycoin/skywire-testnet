package app2

import (
	"context"
	"net"
	"net/rpc"
	"testing"

	"github.com/pkg/errors"
	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/nettest"

	"github.com/skycoin/skywire/pkg/app2/network"
	"github.com/skycoin/skywire/pkg/routing"
)

func TestRPCClient_Dial(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		s := prepRPCServer(t, prepGateway())
		rpcL, lisCleanup := prepListener(t)
		defer lisCleanup()
		go s.Accept(rpcL)

		cl := prepClient(t, rpcL.Addr().Network(), rpcL.Addr().String())

		remoteNet := network.TypeDMSG
		remotePK, _ := cipher.GenerateKeyPair()
		remotePort := routing.Port(100)
		remote := network.Addr{
			Net:    remoteNet,
			PubKey: remotePK,
			Port:   remotePort,
		}

		localPK, _ := cipher.GenerateKeyPair()
		dmsgLocal := dmsg.Addr{
			PK:   localPK,
			Port: 101,
		}
		dmsgRemote := dmsg.Addr{
			PK:   remotePK,
			Port: uint16(remotePort),
		}

		dialCtx := context.Background()
		dialConn := dmsg.NewTransport(&MockConn{}, logging.MustGetLogger("dmsg_tp"),
			dmsgLocal, dmsgRemote, 0, func() {})
		var noErr error

		n := &network.MockNetworker{}
		n.On("DialContext", dialCtx, remote).Return(dialConn, noErr)

		network.ClearNetworkers()
		err := network.AddNetworker(remoteNet, n)
		require.NoError(t, err)

		connID, localPort, err := cl.Dial(remote)
		require.NoError(t, err)
		require.Equal(t, connID, uint16(1))
		require.Equal(t, localPort, routing.Port(dmsgLocal.Port))

	})

	t.Run("dial error", func(t *testing.T) {
		s := prepRPCServer(t, prepGateway())
		rpcL, lisCleanup := prepListener(t)
		defer lisCleanup()
		go s.Accept(rpcL)

		cl := prepClient(t, rpcL.Addr().Network(), rpcL.Addr().String())

		remoteNet := network.TypeDMSG
		remotePK, _ := cipher.GenerateKeyPair()
		remotePort := routing.Port(100)
		remote := network.Addr{
			Net:    remoteNet,
			PubKey: remotePK,
			Port:   remotePort,
		}

		dialCtx := context.Background()
		var dialConn net.Conn
		dialErr := errors.New("dial error")

		n := &network.MockNetworker{}
		n.On("DialContext", dialCtx, remote).Return(dialConn, dialErr)

		network.ClearNetworkers()
		err := network.AddNetworker(remoteNet, n)
		require.NoError(t, err)

		connID, localPort, err := cl.Dial(remote)
		require.Error(t, err)
		require.Equal(t, err.Error(), dialErr.Error())
		require.Equal(t, connID, uint16(0))
		require.Equal(t, localPort, routing.Port(0))
	})
}

func TestRPCClient_Listen(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		s := prepRPCServer(t, prepGateway())
		rpcL, lisCleanup := prepListener(t)
		defer lisCleanup()
		go s.Accept(rpcL)

		cl := prepClient(t, rpcL.Addr().Network(), rpcL.Addr().String())

		localNet := network.TypeDMSG
		localPK, _ := cipher.GenerateKeyPair()
		localPort := routing.Port(100)
		local := network.Addr{
			Net:    localNet,
			PubKey: localPK,
			Port:   localPort,
		}

		listenCtx := context.Background()
		var listenLis net.Listener
		var noErr error

		n := &network.MockNetworker{}
		n.On("ListenContext", listenCtx, local).Return(listenLis, noErr)

		network.ClearNetworkers()
		err := network.AddNetworker(localNet, n)
		require.NoError(t, err)

		lisID, err := cl.Listen(local)
		require.NoError(t, err)
		require.Equal(t, lisID, uint16(1))
	})

	t.Run("listen error", func(t *testing.T) {
		s := prepRPCServer(t, prepGateway())
		rpcL, lisCleanup := prepListener(t)
		defer lisCleanup()
		go s.Accept(rpcL)

		cl := prepClient(t, rpcL.Addr().Network(), rpcL.Addr().String())

		localNet := network.TypeDMSG
		localPK, _ := cipher.GenerateKeyPair()
		localPort := routing.Port(100)
		local := network.Addr{
			Net:    localNet,
			PubKey: localPK,
			Port:   localPort,
		}

		listenCtx := context.Background()
		var listenLis net.Listener
		listenErr := errors.New("listen error")

		n := &network.MockNetworker{}
		n.On("ListenContext", listenCtx, local).Return(listenLis, listenErr)

		network.ClearNetworkers()
		err := network.AddNetworker(localNet, n)
		require.NoError(t, err)

		lisID, err := cl.Listen(local)
		require.Error(t, err)
		require.Equal(t, err.Error(), listenErr.Error())
		require.Equal(t, lisID, uint16(0))
	})
}

func TestRPCClient_Accept(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		gateway := prepGateway()

		localPK, _ := cipher.GenerateKeyPair()
		localPort := uint16(100)
		dmsgLocal := dmsg.Addr{
			PK:   localPK,
			Port: localPort,
		}
		remotePK, _ := cipher.GenerateKeyPair()
		remotePort := uint16(101)
		dmsgRemote := dmsg.Addr{
			PK:   remotePK,
			Port: remotePort,
		}
		lisConn := dmsg.NewTransport(&MockConn{}, logging.MustGetLogger("dmsg_tp"),
			dmsgLocal, dmsgRemote, 0, func() {})
		var noErr error

		lis := &MockListener{}
		lis.On("Accept").Return(lisConn, noErr)

		lisID := uint16(1)

		_, err := gateway.lm.add(lisID, lis)
		require.NoError(t, err)

		s := prepRPCServer(t, gateway)
		rpcL, lisCleanup := prepListener(t)
		defer lisCleanup()
		go s.Accept(rpcL)

		cl := prepClient(t, rpcL.Addr().Network(), rpcL.Addr().String())

		wantRemote := network.Addr{
			Net:    network.TypeDMSG,
			PubKey: remotePK,
			Port:   routing.Port(remotePort),
		}

		connID, remote, err := cl.Accept(lisID)
		require.NoError(t, err)
		require.Equal(t, connID, uint16(1))
		require.Equal(t, remote, wantRemote)
	})

	t.Run("accept error", func(t *testing.T) {
		gateway := prepGateway()

		var lisConn net.Conn
		listenErr := errors.New("accept error")

		lis := &MockListener{}
		lis.On("Accept").Return(lisConn, listenErr)

		lisID := uint16(1)

		_, err := gateway.lm.add(lisID, lis)
		require.NoError(t, err)

		s := prepRPCServer(t, gateway)
		rpcL, lisCleanup := prepListener(t)
		defer lisCleanup()
		go s.Accept(rpcL)

		cl := prepClient(t, rpcL.Addr().Network(), rpcL.Addr().String())

		connID, remote, err := cl.Accept(lisID)
		require.Error(t, err)
		require.Equal(t, err.Error(), listenErr.Error())
		require.Equal(t, connID, uint16(0))
		require.Equal(t, remote, network.Addr{})
	})
}

func TestRPCClient_Write(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		gateway := prepGateway()

		writeBuf := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
		writeN := 10
		var noErr error

		conn := &MockConn{}
		conn.On("Write", writeBuf).Return(writeN, noErr)

		connID := uint16(1)

		_, err := gateway.cm.add(connID, conn)
		require.NoError(t, err)

		s := prepRPCServer(t, gateway)
		rpcL, lisCleanup := prepListener(t)
		defer lisCleanup()
		go s.Accept(rpcL)

		cl := prepClient(t, rpcL.Addr().Network(), rpcL.Addr().String())

		n, err := cl.Write(connID, writeBuf)
		require.NoError(t, err)
		require.Equal(t, n, writeN)
	})

	t.Run("write error", func(t *testing.T) {
		gateway := prepGateway()

		writeBuf := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
		writeN := 0
		writeErr := errors.New("write error")

		conn := &MockConn{}
		conn.On("Write", writeBuf).Return(writeN, writeErr)

		connID := uint16(1)

		_, err := gateway.cm.add(connID, conn)
		require.NoError(t, err)

		s := prepRPCServer(t, gateway)
		rpcL, lisCleanup := prepListener(t)
		defer lisCleanup()
		go s.Accept(rpcL)

		cl := prepClient(t, rpcL.Addr().Network(), rpcL.Addr().String())

		n, err := cl.Write(connID, writeBuf)
		require.Error(t, err)
		require.Equal(t, err.Error(), writeErr.Error())
		require.Equal(t, n, 0)
	})
}

func TestRPCClient_Read(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		gateway := prepGateway()

		readBufLen := 10
		readBuf := make([]byte, readBufLen)
		readN := 5
		var noErr error

		conn := &MockConn{}
		conn.On("Read", readBuf).Return(readN, noErr)

		connID := uint16(1)

		_, err := gateway.cm.add(connID, conn)
		require.NoError(t, err)

		s := prepRPCServer(t, gateway)
		rpcL, lisCleanup := prepListener(t)
		defer lisCleanup()
		go s.Accept(rpcL)

		cl := prepClient(t, rpcL.Addr().Network(), rpcL.Addr().String())

		n, err := cl.Read(connID, readBuf)
		require.NoError(t, err)
		require.Equal(t, n, readN)
	})

	t.Run("read error", func(t *testing.T) {
		gateway := prepGateway()

		readBufLen := 10
		readBuf := make([]byte, readBufLen)
		readN := 0
		readErr := errors.New("read error")

		conn := &MockConn{}
		conn.On("Read", readBuf).Return(readN, readErr)

		connID := uint16(1)

		_, err := gateway.cm.add(connID, conn)
		require.NoError(t, err)

		s := prepRPCServer(t, gateway)
		rpcL, lisCleanup := prepListener(t)
		defer lisCleanup()
		go s.Accept(rpcL)

		cl := prepClient(t, rpcL.Addr().Network(), rpcL.Addr().String())

		n, err := cl.Read(connID, readBuf)
		require.Error(t, err)
		require.Equal(t, err.Error(), readErr.Error())
		require.Equal(t, n, readN)
	})
}

func TestRPCClient_CloseConn(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		gateway := prepGateway()

		var noErr error

		conn := &MockConn{}
		conn.On("Close").Return(noErr)

		connID := uint16(1)

		_, err := gateway.cm.add(connID, conn)
		require.NoError(t, err)

		s := prepRPCServer(t, gateway)
		rpcL, lisCleanup := prepListener(t)
		defer lisCleanup()
		go s.Accept(rpcL)

		cl := prepClient(t, rpcL.Addr().Network(), rpcL.Addr().String())

		err = cl.CloseConn(connID)
		require.NoError(t, err)
	})

	t.Run("close error", func(t *testing.T) {
		gateway := prepGateway()

		closeErr := errors.New("close error")

		conn := &MockConn{}
		conn.On("Close").Return(closeErr)

		connID := uint16(1)

		_, err := gateway.cm.add(connID, conn)
		require.NoError(t, err)

		s := prepRPCServer(t, gateway)
		rpcL, lisCleanup := prepListener(t)
		defer lisCleanup()
		go s.Accept(rpcL)

		cl := prepClient(t, rpcL.Addr().Network(), rpcL.Addr().String())

		err = cl.CloseConn(connID)
		require.Error(t, err)
		require.Equal(t, err.Error(), closeErr.Error())
	})
}

func TestRPCClient_CloseListener(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		gateway := prepGateway()

		var noErr error

		lis := &MockListener{}
		lis.On("Close").Return(noErr)

		lisID := uint16(1)

		_, err := gateway.lm.add(lisID, lis)
		require.NoError(t, err)

		s := prepRPCServer(t, gateway)
		rpcL, lisCleanup := prepListener(t)
		defer lisCleanup()
		go s.Accept(rpcL)

		cl := prepClient(t, rpcL.Addr().Network(), rpcL.Addr().String())

		err = cl.CloseListener(lisID)
		require.NoError(t, err)
	})

	t.Run("close error", func(t *testing.T) {
		gateway := prepGateway()

		closeErr := errors.New("close error")

		lis := &MockListener{}
		lis.On("Close").Return(closeErr)

		lisID := uint16(1)

		_, err := gateway.lm.add(lisID, lis)
		require.NoError(t, err)

		s := prepRPCServer(t, gateway)
		rpcL, lisCleanup := prepListener(t)
		defer lisCleanup()
		go s.Accept(rpcL)

		cl := prepClient(t, rpcL.Addr().Network(), rpcL.Addr().String())

		err = cl.CloseListener(lisID)
		require.Error(t, err)
		require.Equal(t, err.Error(), closeErr.Error())
	})
}

func prepGateway() *RPCGateway {
	l := logging.MustGetLogger("rpc_gateway")
	return newRPCGateway(l)
}

func prepRPCServer(t *testing.T, gateway *RPCGateway) *rpc.Server {
	s := rpc.NewServer()
	err := s.Register(gateway)
	require.NoError(t, err)

	return s
}

func prepListener(t *testing.T) (lis net.Listener, cleanup func()) {
	lis, err := nettest.NewLocalListener("tcp")
	require.NoError(t, err)

	return lis, func() {
		err := lis.Close()
		require.NoError(t, err)
	}
}

func prepClient(t *testing.T, network, addr string) RPCClient {
	rpcCl, err := rpc.Dial(network, addr)
	require.NoError(t, err)

	return NewRPCClient(rpcCl)
}
