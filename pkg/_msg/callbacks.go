package messaging

type (
	// HandshakeCompleteAction triggers when a handshake is completed successfully.
	HandshakeCompleteAction func(conn *Link)

	// FrameAction triggers when a connection receives a non-predefined frame.
	// If an error is returned, the connection is closed and the 'Close' callback is triggered.
	FrameAction func(conn *Link, dt FrameType, body []byte) error

	// TCPCloseAction triggers when we receive a connection closure.
	// 'remote' determines whether the closure is requested remotely or locally.
	TCPCloseAction func(conn *Link, remote bool)

	// Callbacks contains callbacks.
	Callbacks struct {
		HandshakeComplete HandshakeCompleteAction
		Data              FrameAction
		Close             TCPCloseAction
	}
)
