package router

import (
	"github.com/skycoin/skywire/internal/appnet"
	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/app"
)

// respondFunc is for allowing the separation of;
//  - operations that should be under the protection of sync.RWMutex
//  - operations that don't need to be under the protection of sync.RWMutex
// For example, if we define a function as `func action() respondFunc`;
//  1. logic in 'action' can be protected by sync.RWMutex.
//  2. the returned 'respondFunc' can be made to not be under the same protection.
type respondFunc func() ([]byte, error)

// returns a 'respondFunc' that always returns the given error.
func failWith(err error) respondFunc {
	return func() ([]byte, error) { return nil, err }
}

func makeDataHandlers(ap *AppProc) appnet.HandlerMap {
	return appnet.HandlerMap{
		appnet.FrameCreateLoop: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			var rAddr app.LoopAddr
			if err := rAddr.Decode(b); err != nil {
				return nil, err
			}
			return handleRequestLoop(ap, rAddr)()
		},
		appnet.FrameCloseLoop: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			var lm app.LoopMeta
			if err := lm.Decode(b); err != nil {
				return nil, err
			}
			return handleCloseLoop(ap, lm)()
		},
		appnet.FrameData: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			var df app.DataFrame
			if err := df.Decode(b); err != nil {
				return nil, err
			}
			return handleDataFrame(ap, df.Meta, df.Data)()
		},
	}
}

// triggered when App sends 'CreateLoop' frame to Host
func handleRequestLoop(ap *AppProc, rAddr app.LoopAddr) respondFunc {
	ap.mx.Lock()
	defer ap.mx.Unlock()

	// prepare noise
	ns, err := noise.KKAndSecp256k1(noise.Config{
		LocalPK:   ap.e.Config().HostPK,
		LocalSK:   ap.e.Config().HostSK,
		RemotePK:  rAddr.PubKey,
		Initiator: true,
	})
	if err != nil {
		return failWith(err)
	}
	msg, err := ns.HandshakeMessage()
	if err != nil {
		return failWith(err)
	}

	// allocate local listening port for the new loop
	lPort := ap.pm.AllocPort(ap.pid)

	lm := app.LoopMeta{
		Local:  app.LoopAddr{PubKey: ap.e.Config().HostPK, Port: lPort},
		Remote: rAddr,
	}

	// keep track of the new loop (if not already exists)
	if _, ok := ap.lps[lm]; ok {
		return failWith(ErrLoopAlreadyExists)
	}
	ap.lps[lm] = &loopDispatch{ns: ns}

	// if loop is of loopback type (dst app is on localhost) send to local app, else send to router.
	if lm.IsLoopback() {
		return func() ([]byte, error) {
			a2, ok := ap.pm.ProcOfPort(lm.Remote.Port)
			if !ok {
				return nil, ErrProcNotFound
			}
			_, err := a2.e.Call(appnet.FrameConfirmLoop, lm.Swap().Encode())
			return lm.Encode(), err
		}
	}
	return func() ([]byte, error) {
		return lm.Encode(), ap.r.FindRoutesAndSetupLoop(lm, msg)
	}
}

// triggered when App sends 'CloseLoop' frame to Host
func handleCloseLoop(ap *AppProc, lm app.LoopMeta) respondFunc {
	ap.mx.Lock()
	delete(ap.lps, lm)
	ap.mx.Unlock()

	if lm.IsLoopback() {
		return func() ([]byte, error) {
			a2, ok := ap.pm.ProcOfPort(lm.Remote.Port)
			if !ok {
				return nil, ErrProcNotFound
			}
			_, err := a2.e.Call(appnet.FrameCloseLoop, lm.Encode())
			return nil, err
		}
	}
	return func() ([]byte, error) {
		return nil, ap.r.CloseLoop(lm)
	}
}

// triggered when App sends 'Data' frame to Host
func handleDataFrame(ap *AppProc, lm app.LoopMeta, plaintext []byte) respondFunc {
	if lm.IsLoopback() {
		return func() ([]byte, error) {
			rA, ok := ap.pm.ProcOfPort(lm.Remote.Port)
			if !ok {
				return nil, ErrLoopNotFound
			}
			df := app.DataFrame{Meta: *lm.Swap(), Data: plaintext}
			_, err := rA.e.Call(appnet.FrameData, df.Encode())
			return nil, err
		}
	}
	ap.mx.RLock()
	ld, ok := ap.lps[lm]
	ap.mx.RUnlock()
	if !ok {
		return failWith(ErrLoopNotFound)
	}
	return func() ([]byte, error) {
		return nil, ap.r.ForwardPacket(ld.trID, ld.rtID, ld.ns.Encrypt(plaintext))
	}
}

func makeCtrlHandlers(_ *AppProc) appnet.HandlerMap {
	// TODO(evanlinjin): implement.
	return appnet.HandlerMap{}
}
