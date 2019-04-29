package router

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/app/apptest"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/routing"
)

type mockRouter struct {
	done chan struct{}
}

func newMockRouter() *mockRouter {
	return &mockRouter{done: make(chan struct{})}
}

func (mr *mockRouter) Serve(ctx context.Context, _ ProcManager) error {
	select {
	case <-ctx.Done():
	case <-mr.done:
	}
	return nil
}

func (*mockRouter) FindRoutesAndSetupLoop(app.LoopMeta, []byte) error {
	return nil
}

func (*mockRouter) ForwardPacket(uuid.UUID, routing.RouteID, []byte) error {
	return nil
}

func (*mockRouter) CloseLoop(app.LoopMeta) error {
	return nil
}

func (mr *mockRouter) Close() error {
	close(mr.done)
	return nil
}

func TestProcManager_RunProc(t *testing.T) {

	// All test binary files should be contained within this temp dir.
	// The entire dir will be deleted after tests.
	tempDir, err := ioutil.TempDir(os.TempDir(), "sw_test_")
	require.NoError(t, err)
	defer func() { require.NoError(t, os.RemoveAll(tempDir)) }()

	// Mock router stores no internal state and can be reused between tests.
	r := newMockRouter()
	defer func() { require.NoError(t, r.Close()) }()

	// Pre-determined key pair for the host node.
	hPK, hSK := cipher.GenerateKeyPair()

	// ProcManager.RunProc() should fail with invalid binLoc.
	// The returned error should contain "invalid binary file".
	t.Run("c", func(t *testing.T) {

		invalidLoc := filepath.Join(tempDir, "app0")

		meta := app.Meta{
			AppName:         "app0",
			AppVersion:      "1.0",
			ProtocolVersion: app.ProtocolVersion,
			Host:            hPK,
		}

		conf := app.ExecConfig{
			HostPK:  hPK,
			HostSK:  hSK,
			WorkDir: ".",
			BinLoc:  invalidLoc,
		}

		pm := NewProcManager(10)

		proc, err := pm.RunProc(r, 1, &meta, &conf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid binary file")
		assert.Nil(t, proc)
	})

	// ProcManager.RunProc() should succeeded with valid values.
	// ProcIDs of subsequent procs should increment starting from 1.
	// We should be able to obtain the started proc with ProcManager.Proc() and ProcManager.ProcOfPort().
	// The returned proc should return false on AppProc.Stopped().
	// The returned proc should stop after AppProc.Stop() is called.
	t.Run("MustSucceedWithValidValues", func(t *testing.T) {

		pm := NewProcManager(10)

		cases := []struct {
			AppName string
			Port    uint16
		}{
			{AppName: "app0", Port: 4},
			{AppName: "app1", Port: 5},
			{AppName: "app2", Port: 34},
		}

		for i, c := range cases {

			meta := app.Meta{
				AppName:         c.AppName,
				AppVersion:      "1.0",
				ProtocolVersion: app.ProtocolVersion,
				Host:            hPK,
			}

			binLoc := filepath.Join(tempDir, c.AppName)
			require.NoError(t, apptest.GenerateMockApp(binLoc, meta), i)

			conf := app.ExecConfig{
				HostPK:  hPK,
				HostSK:  hSK,
				WorkDir: ".",
				BinLoc:  binLoc,
			}

			proc, err := pm.RunProc(r, c.Port, &meta, &conf)
			require.NoError(t, err)
			require.NotNil(t, proc)

			// the proc's assigned pid is expected to increment with
			// subsequent calls to ProcManager.RunProc() and starts from pid of 1
			expPID := ProcID(i + 1)
			assert.Equal(t, expPID, proc.ProcID())

			// should be able to obtain proc from it's pid
			procFromPID, ok := pm.Proc(expPID)
			assert.True(t, ok)
			assert.Equal(t, proc, procFromPID)

			// should be able to obtain proc from it's assigned port
			procFromPort, ok := pm.ProcOfPort(c.Port)
			assert.True(t, ok)
			assert.Equal(t, proc, procFromPort)

			// proc should be running
			assert.False(t, proc.Stopped())

			// should stop successfully
			assert.NoError(t, proc.Stop())
			assert.True(t, proc.Stopped())
		}
	})

	// Subsequent calls to ProcManager.RunProc() should assign expected proc IDs.
	// ProcIDs should be assigned starting from 1 and increment with every subsequent call to ProcManager.RunProc(),
	// even when the proc fails to start.
	// A proc that fails to start, should not be obtainable via ProcManager.Proc() or ProcManager.ProcOfPort().
	// TODO(evanlinjin): Implement.
	t.Run("MustAssignCorrectProcIDs", func(t *testing.T) {

		cases := []struct {
			AppName  string
			Port     uint16
			ValidApp bool // if set to false, test should use a non-existent binLoc to ensure .RunProc() fails.
		}{
			{},
		}

		t.Skip("Not implemented!", cases)
	})

	// Calls to ProcManager.RunProc() should fail if the 'port' input contains a port that is already reserved.
	// TODO(evanlinjin): Implement.
	t.Run("MustFailWhenUsedPortIsAssigned", func(t *testing.T) {
		t.Skip("Not implemented!")
	})
}

// ProcManager.AllocPort() must fail when allocating a used port.
// ProcManager.AllocPort() must increment the port on each call (even on unsuccessful allocations).
// ProcManager.AllocPort() must start allocating ports from the specified minPort.
// TODO(evanlinjin): Implement.
func TestProcManager_AllocPort(t *testing.T) {
	t.Skip("Not implemented!")
}

// ProcManager.RangeProcIDs() should only range running procs.
// TODO(evanlinjin): Implement.
func TestProcManager_RangeProcIDs(t *testing.T) {
	t.Skip("Not implemented!")
}

// ProcManager.RangePorts() should only range running procs.
// TODO(evanlinjin): Implement.
func TestProcManager_RangePorts(t *testing.T) {
	t.Skip("Not implemented!")
}

// ProcManager.Close() should stop all processes.
// - Any call to .Proc() and .ProcOfPort() should return false.
// - Any call to .RangeProcIDs() and .RangePorts() should range through zero results.
// TODO(evanlinjin): Implement.
func TestProcManager_Close(t *testing.T) {
	t.Skip("Not implemented!")
}
