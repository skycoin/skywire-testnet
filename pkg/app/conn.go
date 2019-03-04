package app

import (
	"fmt"
	"net"
	"os"
	"time"
)

// PipeAddr implements net.Addr for PipeConn.
type PipeAddr struct {
	pipePath string
}

// Network returns custom pipe Network type.
func (pa *PipeAddr) Network() string {
	return "pipe"
}

func (pa *PipeAddr) String() string {
	return pa.pipePath
}

// PipeConn implements net.Conn interface over a pair of unix pipes.
type PipeConn struct {
	inFile  *os.File
	outFile *os.File
}

// OpenPipeConn creates a pair of unix pipe and setups PipeConn over
// that pair.
func OpenPipeConn() (srvConn *PipeConn, clientConn *PipeConn, err error) {
	srvIn, clientOut, err := os.Pipe()
	if err != nil {
		err = fmt.Errorf("failed to open server pipe: %s", err)
		return
	}

	clientIn, srvOut, err := os.Pipe()
	if err != nil {
		err = fmt.Errorf("failed to open client pipe: %s", err)
		return
	}

	clientConn = &PipeConn{clientIn, clientOut}
	srvConn = &PipeConn{srvIn, srvOut}
	return srvConn, clientConn, err
}

// NewPipeConn constructs new PipeConn from already opened pipe fds.
func NewPipeConn(inFd, outFd uintptr) (*PipeConn, error) {
	inFile := os.NewFile(inFd, "|0")
	if _, err := inFile.Stat(); os.IsNotExist(err) {
		return nil, fmt.Errorf("inFile does not exist")
	}

	outFile := os.NewFile(outFd, "|1")
	if _, err := outFile.Stat(); os.IsNotExist(err) {
		return nil, fmt.Errorf("outFile does not exist")
	}

	return &PipeConn{inFile, outFile}, nil
}

func (conn *PipeConn) Read(b []byte) (n int, err error) {
	return conn.inFile.Read(b)
}

func (conn *PipeConn) Write(b []byte) (n int, err error) {
	return conn.outFile.Write(b)
}

// Close closes the connection.
func (conn *PipeConn) Close() error {
	inErr := conn.inFile.Close()
	outErr := conn.outFile.Close()
	if inErr != nil {
		return fmt.Errorf("failed to close input pipe: %s", inErr)
	}

	if outErr != nil {
		return fmt.Errorf("failed to close output pipe: %s", outErr)
	}

	return nil
}

// LocalAddr returns the local network address.
func (conn *PipeConn) LocalAddr() net.Addr {
	return &PipeAddr{conn.inFile.Name()}
}

// RemoteAddr returns the remote network address.
func (conn *PipeConn) RemoteAddr() net.Addr {
	return &PipeAddr{conn.outFile.Name()}
}

// SetDeadline implements the Conn SetDeadline method.
func (conn *PipeConn) SetDeadline(t time.Time) error {
	if err := conn.inFile.SetDeadline(t); err != nil {
		return fmt.Errorf("failed to set input pipe deadline: %s", err)
	}

	if err := conn.outFile.SetDeadline(t); err != nil {
		return fmt.Errorf("failed to set out pipe deadline: %s", err)
	}

	return nil
}

// SetReadDeadline implements the Conn SetReadDeadline method.
func (conn *PipeConn) SetReadDeadline(t time.Time) error {
	return conn.inFile.SetDeadline(t)
}

// SetWriteDeadline implements the Conn SetWriteDeadline method.
func (conn *PipeConn) SetWriteDeadline(t time.Time) error {
	return conn.outFile.SetDeadline(t)
}

// Fd returns file descriptors for a pipe pair
func (conn *PipeConn) Fd() (inFd uintptr, outFd uintptr) {
	inFd = conn.inFile.Fd()
	outFd = conn.outFile.Fd()
	return
}
