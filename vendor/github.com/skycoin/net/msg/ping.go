package msg

import (
	"encoding/binary"
	"time"
)

const (
	PING_MSG_TIME_SIZE = 8
)

const (
	PING_MSG_HEADER_BEGIN = 0
	PING_MSG_TYPE_BEGIN
	PING_MSG_TYPE_END = MSG_TYPE_BEGIN + MSG_TYPE_SIZE
	PING_MSG_TIME_BEGIN
	PING_MSG_TIME_END = PING_MSG_TIME_BEGIN + PING_MSG_TIME_SIZE
	PING_MSG_HEADER_END
	PING_MSG_HEADER_SIZE
)

func unixMillisecond() uint64 {
	now := time.Now()
	sec := now.Unix() * 1000
	m := now.Nanosecond() / 1e6
	return uint64(sec + int64(m))
}

func GenPingMsg() []byte {
	b := make([]byte, PING_MSG_HEADER_SIZE)
	b[PING_MSG_TYPE_BEGIN] = TYPE_PING
	binary.BigEndian.PutUint64(b[PING_MSG_TIME_BEGIN:], unixMillisecond())
	return b
}
