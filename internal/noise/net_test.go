package noise

import (
	"fmt"
	"io"
	"net"
	"net/rpc"
	"sync"
	"testing"
	"time"

	"github.com/flynn/noise"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
)

type TestRPC struct{}

type AddIn struct{ A, B int }

func (r *TestRPC) Add(in *AddIn, out *int) error {
	*out = in.A + in.B
	return nil
}

func TestRPCClientDialer(t *testing.T) {
	var (
		pattern = HandshakeXK
	)

	svr := rpc.NewServer()
	require.NoError(t, svr.Register(new(TestRPC)))

	lPK, lSK := cipher.GenerateKeyPair()
	var l net.Listener
	var lAddr string

	setup := func() {
		if len(lAddr) == 0 {
			lAddr = ":0"
		}
		var err error

		l, err = net.Listen("tcp", lAddr)
		require.NoError(t, err)

		l = WrapListener(l, lPK, lSK, false, pattern)
		lAddr = l.Addr().(*Addr).Addr.String()
		t.Logf("Listening on %s", lAddr)
	}

	teardown := func() {
		if l != nil {
			require.NoError(t, l.Close())
			l = nil
		}
	}

	t.Run("RunRetry", func(t *testing.T) {
		setup()
		defer teardown() // Just in case of failure.

		const reconCount = 5
		const retry = time.Second / 4

		dPK, dSK := cipher.GenerateKeyPair()
		d := NewRPCClientDialer(lAddr, pattern, Config{
			LocalPK:   dPK,
			LocalSK:   dSK,
			RemotePK:  lPK,
			Initiator: true,
		})
		dDone := make(chan error, 1)

		go func() {
			dDone <- d.Run(svr, retry)
			close(dDone)
		}()

		for i := 0; i < reconCount; i++ {
			teardown()
			time.Sleep(retry * 2) // Dialer shouldn't quit retrying in this time.
			setup()

			conn, err := l.Accept()
			require.NoError(t, err)

			in, out := &AddIn{A: i, B: i}, new(int)
			require.NoError(t, rpc.NewClient(conn).Call("TestRPC.Add", in, out))
			require.Equal(t, in.A+in.B, *out)
			require.NoError(t, conn.Close())
		}

		_ = d.Close()
		require.NoError(t, <-dDone)
	})
}

func TestConn(t *testing.T) {
	type Result struct {
		N   int
		Err error
	}

	const timeout = time.Second

	aPK, aSK := cipher.GenerateKeyPair()
	bPK, bSK := cipher.GenerateKeyPair()

	aNs, err := XKAndSecp256k1(Config{LocalPK: aPK, LocalSK: aSK, RemotePK: bPK, Initiator: true})
	require.NoError(t, err)
	bNs, err := XKAndSecp256k1(Config{LocalPK: bPK, LocalSK: bSK, Initiator: false})
	require.NoError(t, err)

	aConn, bConn := net.Pipe()
	defer func() { _, _ = aConn.Close(), bConn.Close() }()

	aRW := NewReadWriter(aConn, aNs)
	bRW := NewReadWriter(bConn, bNs)

	errChan := make(chan error, 2)
	go func() { errChan <- aRW.Handshake(timeout) }()
	go func() { errChan <- bRW.Handshake(timeout) }()
	require.NoError(t, <-errChan)
	require.NoError(t, <-errChan)
	close(errChan)

	a := &Conn{Conn: aConn, ns: aRW}
	b := &Conn{Conn: bConn, ns: bRW}

	t.Run("ReadWrite", func(t *testing.T) {
		aResults := make(chan Result)
		bResults := make(chan Result)

		for i := 0; i < 10; i++ {
			msgAtoB := []byte(fmt.Sprintf("this is message %d from A for B", i))

			go func() {
				n, err := a.Write(msgAtoB)
				aResults <- Result{N: n, Err: err}
			}()

			receivedMsgAtoB := make([]byte, len(msgAtoB))
			n, err := io.ReadFull(b, receivedMsgAtoB)
			require.Equal(t, len(msgAtoB), n)
			require.NoError(t, err)

			aResult := <-aResults
			require.Equal(t, len(msgAtoB), aResult.N)
			require.NoError(t, aResult.Err)

			msgBtoA := []byte(fmt.Sprintf("this is message %d from B for A", i))

			go func() {
				n, err := b.Write(msgAtoB)
				bResults <- Result{N: n, Err: err}
			}()

			receivedMsgBtoA := make([]byte, len(msgBtoA))
			n, err = io.ReadFull(a, receivedMsgBtoA)
			require.Equal(t, len(msgBtoA), n)
			require.NoError(t, err)

			bResult := <-bResults
			require.Equal(t, len(msgBtoA), bResult.N)
			require.NoError(t, bResult.Err)
		}
	})

	t.Run("ReadWriteConcurrent", func(t *testing.T) {
		type ReadResult struct {
			Msg string
			N   int
			Err error
		}
		const (
			MsgCount = 100
			MsgLen   = 4
		)
		var (
			aMap    = make(map[string]struct{})
			bMap    = make(map[string]struct{})
			aWrites = make(chan Result, MsgCount)
			bWrites = make(chan Result, MsgCount)
			aReads  = make(chan ReadResult, MsgCount)
			bReads  = make(chan ReadResult, MsgCount)
		)
		randSleep := func() { time.Sleep(time.Duration(cipher.RandByte(1)[0]) / 255 * time.Second) }

		for i := 0; i < MsgCount; i++ {
			msg := fmt.Sprintf("%4d", i)
			go func() {
				randSleep()
				n, err := a.Write([]byte(msg))
				aWrites <- Result{N: n, Err: err}
			}()
			go func() {
				randSleep()
				n, err := b.Write([]byte(msg))
				bWrites <- Result{N: n, Err: err}
			}()
			go func() {
				randSleep()
				msg := make([]byte, MsgLen)
				n, err := io.ReadFull(a, msg)
				aReads <- ReadResult{Msg: string(msg), N: n, Err: err}
			}()
			go func() {
				randSleep()
				msg := make([]byte, MsgLen)
				n, err := io.ReadFull(b, msg)
				bReads <- ReadResult{Msg: string(msg), N: n, Err: err}
			}()
		}

		for i := 0; i < MsgCount; i++ {
			aWrite := <-aWrites
			require.NoError(t, aWrite.Err)
			require.Equal(t, MsgLen, aWrite.N)

			bWrite := <-bWrites
			require.NoError(t, bWrite.Err)
			require.Equal(t, MsgLen, bWrite.N)

			aRead := <-aReads
			require.NoError(t, aRead.Err)
			require.Equal(t, MsgLen, aRead.N)
			_, aHas := aMap[aRead.Msg]
			require.False(t, aHas)
			aMap[aRead.Msg] = struct{}{}

			bRead := <-bReads
			require.NoError(t, bRead.Err)
			require.Equal(t, MsgLen, bRead.N)
			_, bHas := bMap[bRead.Msg]
			require.False(t, bHas)
			bMap[bRead.Msg] = struct{}{}
		}

		require.Len(t, aMap, MsgCount)
		require.Len(t, bMap, MsgCount)
	})

	t.Run("ReadWriteIrregular", func(t *testing.T) {
		const segLen = 100
		const segCount = 1000

		aResults := make([]Result, segCount)

		msg := cipher.RandByte(segLen * segCount)

		wg := new(sync.WaitGroup)
		wg.Add(1)
		go func() {
			for i := 0; i < segCount; i++ {
				n, err := a.Write(msg[i*segLen : (i+1)*segLen])
				aResults[i] = Result{N: n, Err: err}
			}
			wg.Done()
		}()

		msgResult := make([]byte, len(msg))
		_, err := io.ReadFull(b, msgResult)
		require.NoError(t, err)
		require.Equal(t, msg, msgResult)

		wg.Wait()

		for i, r := range aResults {
			require.NoError(t, r.Err, i)
			require.Equal(t, segLen, r.N, i)
		}
	})
}

func TestListener(t *testing.T) {
	const (
		connCount = 10
		msg       = "Hello, world!"
		timeout   = time.Second
	)
	var (
		pattern = noise.HandshakeXK
	)

	dialAndWrite := func(remote cipher.PubKey, addr string) error {
		pk, sk := cipher.GenerateKeyPair()
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return err
		}
		ns, err := New(pattern, Config{LocalPK: pk, LocalSK: sk, RemotePK: remote, Initiator: true})
		if err != nil {
			return err
		}
		conn, err = WrapConn(conn, ns, timeout)
		if err != nil {
			return err
		}
		_, err = conn.Write([]byte(msg))
		if err != nil {
			return err
		}
		return conn.Close()
	}

	lPK, lSK := cipher.GenerateKeyPair()
	l, err := net.Listen("tcp", "")
	require.NoError(t, err)
	defer l.Close()

	l = WrapListener(l, lPK, lSK, false, pattern)
	addr := l.Addr().(*Addr)

	t.Run("Accept", func(t *testing.T) {
		hResults := make([]error, connCount)
		wg := new(sync.WaitGroup)
		wg.Add(1)
		go func() {
			for i := 0; i < connCount; i++ {
				hResults[i] = dialAndWrite(lPK, addr.Addr.String())
			}
			wg.Done()
		}()
		for i := 0; i < connCount; i++ {
			lConn, err := l.Accept()
			require.NoError(t, err)
			rec := make([]byte, len(msg))
			n, err := io.ReadFull(lConn, rec)
			log.Printf("Accept('%s'): received: '%s'", lConn.RemoteAddr(), string(rec))
			require.Equal(t, len(msg), n)
			require.NoError(t, err)
			require.NoError(t, lConn.Close())
		}
		wg.Wait()
		for i := 0; i < connCount; i++ {
			require.NoError(t, hResults[i])
		}
	})
}
