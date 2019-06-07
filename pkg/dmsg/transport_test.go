package dmsg

import (
	"context"
	"errors"
	"testing"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"

	"github.com/skycoin/skywire/pkg/cipher"
)

var (
	log = logging.MustGetLogger("dmsg_test")
)

func TestNewTransport(t *testing.T) {
	tr := NewTransport(nil, log, cipher.PubKey{}, cipher.PubKey{}, 0)
	assert.NotNil(t, tr)
}

func TestTransport_close(t *testing.T) {
	tr := NewTransport(nil, log, cipher.PubKey{}, cipher.PubKey{}, 0)

	closed := tr.close()

	t.Run("Valid close() result (1st attempt)", func(t *testing.T) {
		assert.True(t, closed)
	})

	t.Run("Channel closed (1st attempt)", func(t *testing.T) {
		_, ok := <-tr.doneCh
		assert.False(t, ok)
	})

	closed = tr.close()

	t.Run("Valid close() result (2nd attempt)", func(t *testing.T) {
		assert.False(t, closed)
	})

	t.Run("Channel closed (2nd attempt)", func(t *testing.T) {
		_, ok := <-tr.doneCh
		assert.False(t, ok)
	})
}

func TestTransport_awaitResponse(t *testing.T) {
	tr := NewTransport(nil, log, cipher.PubKey{}, cipher.PubKey{}, 0)

	t.Run("Request rejected", func(t *testing.T) {
		go func() {
			tr.doneCh <- struct{}{}
		}()

		err := tr.awaitResponse(context.TODO())
		assert.Equal(t, ErrRequestRejected, err)
	})

	t.Run("Context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		cancel()
		err := tr.awaitResponse(ctx)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("Invalid remote response", func(t *testing.T) {
		go func() {
			tr.readCh <- MakeFrame(RequestType, 1, nil)
		}()

		err := tr.awaitResponse(context.TODO())
		assert.Equal(t, errors.New("invalid remote response"), err)
	})

	t.Run("Valid remote response", func(t *testing.T) {
		go func() {
			tr.readCh <- MakeFrame(AcceptType, 1, nil)
		}()

		err := tr.awaitResponse(context.TODO())
		assert.NoError(t, err)
	})
}
