package dmsg

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"

	"github.com/skycoin/skywire/pkg/cipher"
)

func TestClient(t *testing.T) {
	logger := logging.MustGetLogger("dms_client")

	p1, p2 := net.Pipe()

	var pk1, pk2 cipher.PubKey
	err := pk1.Set("024ec47420176680816e0406250e7156465e4531f5b26057c9f6297bb0303558c7")
	assert.NoError(t, err)
	err = pk2.Set("031b80cd5773143a39d940dc0710b93dcccc262a85108018a7a95ab9af734f8055")
	assert.NoError(t, err)

	conn1 := NewClientConn(logger, p1, pk1, pk2)
	conn2 := NewClientConn(logger, p2, pk2, pk1)

	conn2.nextInitID = randID(false)

	ch1 := make(chan *Transport, acceptChSize)
	ch2 := make(chan *Transport, acceptChSize)

	ctx := context.TODO()

	go func() {
		_ = conn1.Serve(ctx, ch1)
	}()

	go func() {
		_ = conn2.Serve(ctx, ch2)
	}()

	time.Sleep(100 * time.Millisecond)

	tr, err := conn1.DialTransport(ctx, pk2)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = tr.Close()
	assert.NoError(t, err)
}
