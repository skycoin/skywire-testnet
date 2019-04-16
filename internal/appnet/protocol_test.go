package appnet

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	Type0 = FrameType(0)
	Type1 = FrameType(1)
	Type2 = FrameType(2)
)

var ErrExpected = errors.New("this is an expected error")

//type callChannels [3]chan struct{
//	Req []byte
//	Resp <-chan []byte
//}

type chanArray [3]chan []byte

func makeChanArray(cache int) chanArray {
	var ca chanArray
	for i := range ca {
		ca[i] = make(chan []byte, cache)
	}
	return ca
}

func (ca chanArray) close() {
	for _, c := range ca {
		close(c)
	}
}

func newProto(conn *PipeConn, rqChs, rsChs chanArray) (*Protocol, func(t *testing.T)) {
	var (
		proto = NewProtocol(conn)
		errCh = make(chan error)
	)
	makeHandlerFunc := func(t FrameType) HandlerFunc {
		return func(_ *Protocol, b []byte) ([]byte, error) {
			rqChs[t] <- b
			if resp := <-rsChs[t]; resp != nil {
				return resp, nil
			}
			return nil, ErrExpected
		}
	}
	go func() {
		errCh <- proto.Serve(HandlerMap{
			Type0: makeHandlerFunc(Type0),
			Type1: makeHandlerFunc(Type1),
			Type2: makeHandlerFunc(Type2),
		})
	}()
	return proto, func(t *testing.T) {
		require.NoError(t, proto.Close())
		require.NoError(t, <-errCh)
	}
}

func TestNewProtocol(t *testing.T) {
	const n = 1

	rwA, rwB, err := OpenPipeConn()
	require.NoError(t, err)

	rqA, rsA := makeChanArray(n), makeChanArray(n)
	pA, closeA := newProto(rwA, rqA, rsA)
	defer closeA(t)

	rqB, rsB := makeChanArray(n), makeChanArray(n)
	pB, closeB := newProto(rwB, rqB, rsB)
	defer closeB(t)

	t.Run("SingularCalls", func(t *testing.T) {
		type Case struct {
			Type FrameType
			Req  []byte
			Resp []byte
		}
		cases := []Case{
			{Type: Type0, Req: []byte(""), Resp: nil},
			{Type: Type0, Req: []byte(""), Resp: []byte("out")},
			{Type: Type0, Req: []byte("in"), Resp: []byte("")},
			{Type: Type0, Req: []byte("in"), Resp: []byte("out")},

			{Type: Type1, Req: []byte(""), Resp: nil},
			{Type: Type1, Req: []byte(""), Resp: []byte("out")},
			{Type: Type1, Req: []byte("in"), Resp: []byte("")},
			{Type: Type1, Req: []byte("in"), Resp: []byte("out")},

			{Type: Type2, Req: []byte(""), Resp: nil},
			{Type: Type2, Req: []byte(""), Resp: []byte("out")},
			{Type: Type2, Req: []byte("in"), Resp: []byte("")},
			{Type: Type2, Req: []byte("in"), Resp: []byte("out")},
		}

		for _, c := range cases {
			run := func(proto *Protocol, reqChs, respChs chanArray) {
				respChs[c.Type] <- c.Resp
				resp, err := proto.Call(c.Type, c.Req)

				if c.Resp != nil {
					assert.NoError(t, err)
					assert.Equal(t, c.Resp, resp)
				} else {
					require.Error(t, err)
					assert.Equal(t, ErrExpected.Error(), err.Error())
				}
				assert.Equal(t, c.Req, <-reqChs[c.Type])
			}
			run(pA, rqB, rsB)
			run(pB, rqA, rsA)
		}
	})

	t.Run("ParallelCalls", func(t *testing.T) {
		// TODO(evanlinjin): Implement.
	})
}

//func TestProtocolParallel(t *testing.T) {
//	rw1, rw2, err := OpenPipeConn()
//	require.NoError(t, err)
//	proto1 := NewProtocol(rw1)
//	proto2 := NewProtocol(rw2)
//
//	errCh1 := make(chan error)
//	go func() {
//		errCh1 <- proto1.Serve(HandlerMap{
//			FrameCreateLoop: func(p *Protocol, b []byte) ([]byte, error) {
//				return proto1.Call(FrameConfirmLoop, []byte("foo"))
//			},
//		})
//	}()
//
//	errCh2 := make(chan error)
//	go func() {
//		errCh2 <- proto2.ServeJSON(func(f FrameType, _ []byte) (interface{}, error) {
//			if f != FrameConfirmLoop {
//				return nil, errors.New("unexpected frame")
//			}
//
//			return nil, nil
//		})
//	}()
//
//	require.NoError(t, proto2.CallJSON(FrameCreateLoop, "foo", nil))
//
//	require.NoError(t, proto1.Close())
//	require.NoError(t, proto2.Close())
//
//	require.NoError(t, <-errCh1)
//	require.NoError(t, <-errCh2)
//}