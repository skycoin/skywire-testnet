// +build !no_ci

package messaging

import (
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateResponderPool(t *testing.T, msgAction FrameAction) *Pool {
	responder := NewPool(DefaultLinkConfig(), &Callbacks{Data: msgAction})
	pk, sk := cipher.GenerateKeyPair()
	responder.config.Public = pk
	responder.config.Secret = sk

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() {
		err := responder.Respond(l)
		if err != nil && err != ErrPoolClosed {
			t.Log(err)
		}
	}()
	return responder
}

func generateInitiatorPool(msgAction FrameAction) *Pool {
	initiator := NewPool(DefaultLinkConfig(), &Callbacks{Data: msgAction})
	pk, sk := cipher.GenerateKeyPair()
	initiator.config.Public = pk
	initiator.config.Secret = sk
	return initiator
}

func TestNewPool(t *testing.T) {
	const (
		Initiators       = 10
		MsgsPerInitiator = 20
		MsgLen           = 200
	)

	var results [Initiators * MsgsPerInitiator][2][]byte

	wg := new(sync.WaitGroup)
	wg.Add(Initiators * MsgsPerInitiator)

	responder := generateResponderPool(t, func(_ *Link, _ FrameType, msg []byte) error {
		results[binary.BigEndian.Uint32(msg[1:5])][1] = msg[1:]
		wg.Done()
		return nil
	})

	for i := 0; i < Initiators; i++ {
		initiator := generateInitiatorPool(connMessageAction())
		conn, err := net.Dial("tcp", responder.Listener().Addr().String())
		require.NoError(t, err)
		link, err := initiator.Initiate(conn, responder.config.Public)
		require.NoError(t, err)

		go func(i int, initiator *Pool, conn *Link) {
			for j := 0; j < MsgsPerInitiator; j++ {

				msg := cipher.RandByte(MsgLen)

				resultsIdx := i*MsgsPerInitiator + j
				binary.BigEndian.PutUint32(msg[:4], uint32(resultsIdx))

				results[i*MsgsPerInitiator+j][0] = msg

				_, err = conn.Send(1, msg)
				if err != nil {
					t.Error(err)
				}
			}
			initiator.Close()
		}(i, initiator, link)
	}

	wg.Wait()

	for _, result := range results {
		assert.Equal(t, result[0], result[1])
	}

	responder.Close()
}

func TestPool_Range(t *testing.T) {
	const (
		InitiatorCount = 5
	)

	responder := generateResponderPool(t, func(_ *Link, _ FrameType, _ []byte) error { return nil })

	var initiators []*Pool
	for i := 0; i < InitiatorCount; i++ {
		initiator := generateInitiatorPool(connMessageAction())
		conn, err := net.Dial("tcp", responder.Listener().Addr().String())
		require.NoError(t, err)
		_, err = initiator.Initiate(conn, responder.config.Public)
		require.NoError(t, err)
		initiators = append(initiators, initiator)
	}

	time.Sleep(100 * time.Millisecond)

	// Expect 10 connections.
	count := int(0)
	err := responder.Range(func(_ cipher.PubKey, _ *Link) bool {
		count++
		return true
	})

	require.NoError(t, err)
	require.Equal(t, InitiatorCount, count)

	for _, initiator := range initiators {
		initiator.Close()
	}
	responder.Close()
}

func TestPool_All(t *testing.T) {
	const (
		InitiatorCount = 5
	)

	responder := generateResponderPool(t, func(_ *Link, _ FrameType, _ []byte) error { return nil })

	var initiators []*Pool
	for i := 0; i < InitiatorCount; i++ {
		initiator := generateInitiatorPool(connMessageAction())
		conn, err := net.Dial("tcp", responder.Listener().Addr().String())
		require.NoError(t, err)
		_, err = initiator.Initiate(conn, responder.config.Public)
		require.NoError(t, err)
		initiators = append(initiators, initiator)
	}

	time.Sleep(100 * time.Millisecond)

	// Expect 10 connections.
	all := responder.All()
	require.Equal(t, InitiatorCount, len(all))

	for _, initiator := range initiators {
		initiator.Close()
	}
	responder.Close()
}
