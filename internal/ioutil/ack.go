package ioutil

import (
	"bytes"
	"errors"
	"io"
	"math"
	"sync"
	"time"

	"github.com/skycoin/skywire/pkg/cipher"
)

// DataPacketType defines types of data packets.
type DataPacketType byte

const (
	// DataPacketPayload represents Payload data packet.
	DataPacketPayload DataPacketType = iota
	// DataPacketAck represents Ack data packet.
	DataPacketAck
)

// ErrClosed is the error used for read or write operations on a closed ReadWriter.
var ErrClosed = errors.New("read/write: closed")

// AckReadWriter is an io.ReadWriter wrapper that implements ack logic
// for writes. Writes are blocked till Ack packets are received, CRC
// check is performed using SHA256. Ack packets are either sent along
// with subsequent writes or flushed each ackInterval.
type AckReadWriter struct {
	rw io.ReadWriteCloser

	sndAcks *ackList
	rcvAcks *ackList

	readChan    chan []byte
	errChan     chan error
	buf         *bytes.Buffer
	doneChan    chan struct{}
	ackInterval time.Duration
}

// NewAckReadWriter constructs a new AckReadWriter.
func NewAckReadWriter(rw io.ReadWriteCloser, ackInterval time.Duration) *AckReadWriter {
	arw := &AckReadWriter{
		rw:          rw,
		sndAcks:     newAckList(),
		rcvAcks:     newAckList(),
		doneChan:    make(chan struct{}),
		readChan:    make(chan []byte),
		errChan:     make(chan error),
		ackInterval: ackInterval,
		buf:         new(bytes.Buffer),
	}
	go arw.serveLoop()
	return arw
}

func (arw *AckReadWriter) Write(p []byte) (n int, err error) {
	errCh := make(chan error)
	seq := arw.sndAcks.push(&ack{errCh, cipher.SumSHA256(p)})
	packet := append([]byte{byte(DataPacketPayload), seq}, p...)

	_, _, buf := arw.ackPacket()
	buf = append(buf, packet...)
	n, err = arw.rw.Write(buf)
	if err != nil {
		return
	}

	if n != len(buf) {
		err = io.ErrShortWrite
		return
	}

	select {
	case <-arw.doneChan:
		return 0, ErrClosed
	case err = <-errCh:
		return len(p), err
	}
}

func (arw *AckReadWriter) Read(p []byte) (n int, err error) {
	if arw.buf.Len() != 0 {
		return arw.buf.Read(p)
	}

	select {
	case <-arw.doneChan:
		return 0, io.EOF
	case err := <-arw.errChan:
		return 0, err
	case data, more := <-arw.readChan:
		if !more {
			return 0, io.EOF
		}

		time.AfterFunc(arw.ackInterval, arw.flush)

		if len(data) > len(p) {
			if _, err := arw.buf.Write(data[len(p):]); err != nil {
				return 0, io.ErrShortBuffer
			}

			return copy(p, data[:len(p)]), nil
		}

		return copy(p, data), nil
	}
}

// Close implements io.Closer for AckReadWriter.
func (arw *AckReadWriter) Close() error {
	select {
	case <-arw.doneChan:
	default:
		arw.flush()
		close(arw.doneChan)
		close(arw.readChan)
	}

	return arw.rw.Close()
}

func (arw *AckReadWriter) serveLoop() {
	buf := make([]byte, 100*1024)
	for {
		n, err := arw.rw.Read(buf)
		if err != nil {
			select {
			case <-arw.doneChan:
			case arw.errChan <- err:
			}
			return
		}

		data := buf[:n]
		for {
			if len(data) == 0 || DataPacketType(data[0]) == DataPacketPayload {
				break
			}

			arw.confirm(data[1], data[2:34])
			data = data[34:]
		}

		if len(data) == 0 {
			continue
		}

		arw.rcvAcks.set(data[1], &ack{nil, cipher.SumSHA256(data[2:])})
		go func() {
			select {
			case <-arw.doneChan:
			case arw.readChan <- data[2:]:
			}
		}()
	}
}

func (arw *AckReadWriter) ackPacket() ([]byte, []*ack, []byte) {
	buf := make([]byte, 0)
	acks := make([]*ack, 0)
	seqs := make([]byte, 0)
	for {
		seq, ack := arw.rcvAcks.pull()
		if ack == nil {
			break
		}

		buf = append([]byte{byte(DataPacketAck), seq}, ack.hash[:]...)
		acks = append(acks, ack)
		seqs = append(seqs, seq)
	}

	return seqs, acks, buf
}

func (arw *AckReadWriter) confirm(seq byte, hash []byte) {
	ack := arw.sndAcks.remove(seq)
	if ack == nil {
		return
	}

	rcvHash, err := cipher.SHA256FromBytes(hash)
	if err != nil {
		ack.errChan <- err
		return
	}

	if ack.hash != rcvHash {
		ack.errChan <- errors.New("invalid CRC")
		return
	}

	ack.errChan <- nil
}

func (arw *AckReadWriter) flush() {
	seqs, acks, p := arw.ackPacket()
	if len(p) == 0 {
		return
	}

	if _, err := arw.rw.Write(p); err != nil {
		for idx, ack := range acks {
			arw.rcvAcks.set(seqs[idx], ack)
		}
	}
}

type ack struct {
	errChan chan error
	hash    cipher.SHA256
}

type ackList struct {
	sync.Mutex

	acks []*ack
}

func newAckList() *ackList {
	return &ackList{acks: make([]*ack, math.MaxUint8)}
}

func (al *ackList) push(a *ack) byte {
	al.Lock()
	defer al.Unlock()

	for i := byte(0); i < math.MaxUint8; i++ {
		if al.acks[i] == nil {
			al.acks[i] = a
			return i
		}
	}

	panic("too many packets in flight")
}

func (al *ackList) pull() (byte, *ack) {
	al.Lock()
	defer al.Unlock()

	for seq, ack := range al.acks {
		if ack != nil {
			al.acks[seq] = nil
			return byte(seq), ack
		}
	}

	return 0, nil
}

func (al *ackList) set(seq byte, a *ack) {
	al.Lock()
	al.acks[seq] = a
	al.Unlock()
}

func (al *ackList) remove(seq byte) *ack {
	al.Lock()
	a := al.acks[seq]
	al.acks[seq] = nil
	al.Unlock()
	return a
}
