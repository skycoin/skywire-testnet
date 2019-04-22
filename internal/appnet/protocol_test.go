package appnet

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	FrameType0 = FrameType(0)
	FrameType1 = FrameType(1)
	FrameType2 = FrameType(2)
)

var ErrExpected = errors.New("this is an expected error")

type callResponse struct {
	B []byte
	E error
}

type expector interface {
	Set(t FrameType, req []byte, resp callResponse)
	Get(t FrameType, req []byte) callResponse
}

type mapExpector struct {
	sync.Map
}

func (m *mapExpector) Set(t FrameType, req []byte, resp callResponse) {
	key := string(append([]byte{byte(t)}, req...))
	m.Store(key, resp)
}

func (m *mapExpector) Get(t FrameType, req []byte) callResponse {
	key := string(append([]byte{byte(t)}, req...))
	resp, ok := m.Load(key)
	if !ok {
		return callResponse{}
	}
	return resp.(callResponse)
}

func newProto(conn *PipeConn, exp expector) (*Protocol, func(t *testing.T)) {
	var (
		proto = NewProtocol(conn)
		errCh = make(chan error)
	)
	makeHandlerFunc := func(t FrameType) HandlerFunc {
		return func(_ *Protocol, b []byte) ([]byte, error) {
			expResp := exp.Get(t, b)
			return expResp.B, expResp.E
		}
	}
	go func() {
		errCh <- proto.Serve(HandlerMap{
			FrameType0: makeHandlerFunc(FrameType0),
			FrameType1: makeHandlerFunc(FrameType1),
			FrameType2: makeHandlerFunc(FrameType2),
		})
	}()
	return proto, func(t *testing.T) {
		require.NoError(t, proto.Close())
		require.NoError(t, <-errCh)
	}
}

func TestNewProtocol(t *testing.T) {
	rwA, rwB, err := OpenPipeConn()
	require.NoError(t, err)

	exp := expector(new(mapExpector))

	pA, closeA := newProto(rwA, exp)
	defer closeA(t)

	pB, closeB := newProto(rwB, exp)
	defer closeB(t)

	type Case struct {
		Type FrameType
		Req  []byte
		Resp callResponse
	}

	cases := []Case{
		{Type: FrameType0, Req: []byte("dfg"), Resp: callResponse{E: ErrExpected}},
		{Type: FrameType0, Req: []byte("szrggr"), Resp: callResponse{B: []byte("out")}},
		{Type: FrameType0, Req: []byte("indrg"), Resp: callResponse{E: ErrExpected}},
		{Type: FrameType0, Req: []byte("in"), Resp: callResponse{B: []byte("out")}},

		{Type: FrameType1, Req: []byte("zdrgzdg"), Resp: callResponse{E: ErrExpected}},
		{Type: FrameType1, Req: []byte("zdfg"), Resp: callResponse{B: []byte("out")}},
		{Type: FrameType1, Req: []byte("idzfgn"), Resp: callResponse{E: ErrExpected}},
		{Type: FrameType1, Req: []byte("igfgnn"), Resp: callResponse{B: []byte("out")}},

		{Type: FrameType2, Req: []byte("fghfgh"), Resp: callResponse{E: ErrExpected}},
		{Type: FrameType2, Req: []byte("fnfnf"), Resp: callResponse{B: []byte("out")}},
		{Type: FrameType2, Req: []byte("indfg"), Resp: callResponse{E: ErrExpected}},
		{Type: FrameType2, Req: []byte("idfgvn"), Resp: callResponse{B: []byte("out")}},
	}

	for _, c := range cases {
		exp.Set(c.Type, c.Req, c.Resp)
	}

	t.Run("SingularCalls", func(t *testing.T) {
		for _, c := range cases {
			respA, errA := pA.Call(c.Type, c.Req)
			assert.Equal(t, c.Resp.B, respA)
			assert.Equal(t, c.Resp.E, errA)

			respB, errB := pB.Call(c.Type, c.Req)
			assert.Equal(t, c.Resp.B, respB)
			assert.Equal(t, c.Resp.E, errB)
		}
	})

	t.Run("ParallelCalls", func(t *testing.T) {
		type Result struct {
			Case Case
			Resp callResponse
		}

		resACh := make(chan Result)
		defer close(resACh)

		resBCh := make(chan Result)
		defer close(resBCh)

		for _, c := range cases {

			go func(c Case) {
				b, err := pA.Call(c.Type, c.Req)
				resACh <- Result{Case: c, Resp: callResponse{B: b, E: err}}
			}(c)

			go func(c Case) {
				b, err := pA.Call(c.Type, c.Req)
				resBCh <- Result{Case: c, Resp: callResponse{B: b, E: err}}
			}(c)
		}

		for i := 0; i < len(cases); i++ {
			rA := <-resACh
			assert.Equal(t, rA.Case.Resp, rA.Resp)

			rB := <-resBCh
			assert.Equal(t, rB.Case.Resp, rB.Resp)
		}
	})
}
