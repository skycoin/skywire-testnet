package human

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testGrabber struct {
	delay  time.Duration
	text   string
	called int32
}

func (g *testGrabber) GrabInput(_ context.Context, _, _ string, _ []UserOption) (string, error) {
	atomic.StoreInt32(&g.called, 1)
	time.Sleep(g.delay)
	return g.text, nil
}

func TestMultiGrabberAllCalled(t *testing.T) {
	g1 := &testGrabber{
		delay: 5 * time.Millisecond,
		text:  "first",
	}

	g2 := &testGrabber{
		delay: 1 * time.Millisecond,
		text:  "second",
	}

	g3 := &testGrabber{
		delay: 3 * time.Millisecond,
		text:  "third",
	}

	multiGrabber := NewMultiGrabber(g1, g2, g3)
	_, err := multiGrabber.GrabInput(context.Background(), "", "", nil)

	g1Called := atomic.LoadInt32(&g1.called)
	g2Called := atomic.LoadInt32(&g2.called)
	g3Called := atomic.LoadInt32(&g3.called)

	require.NoError(t, err)

	assert.Equal(t, int32(1), g1Called)
	assert.Equal(t, int32(1), g2Called)
	assert.Equal(t, int32(1), g3Called)
}

func TestMultiGrabberFirstResponse(t *testing.T) {
	g1 := &testGrabber{
		delay: 5 * time.Millisecond,
		text:  "first",
	}

	g2 := &testGrabber{
		delay: 1 * time.Millisecond,
		text:  "second",
	}

	g3 := &testGrabber{
		delay: 3 * time.Millisecond,
		text:  "third",
	}

	multiGrabber := NewMultiGrabber(g1, g2, g3)
	result, err := multiGrabber.GrabInput(context.Background(), "", "", nil)

	require.NoError(t, err)
	assert.Equal(t, "second", result)
}
