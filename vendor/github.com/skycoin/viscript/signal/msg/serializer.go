package msg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func Deserialize(msg []byte, obj interface{}) error {
	msg = msg[2:] //pop off prefix byte
	err := encoder.DeserializeRaw(msg, obj)
	return err
}

func MustDeserialize(msg []byte, obj interface{}) {
	msg = msg[2:] //pop off prefix byte
	err := encoder.DeserializeRaw(msg, obj)
	if err != nil {
		log.Fatal("Error with deserialize", err)
	}
}

func Serialize(prefix uint16, obj interface{}) []byte {
	b := encoder.Serialize(obj)
	b1 := make([]byte, 2)
	b1[0] = (uint8)(prefix & 0x00ff)
	b1[1] = (uint8)((prefix & 0xff00) >> 8)
	b2 := append(b1, b...)
	return b2
}

func GetType(message []byte) uint16 {
	var value uint16
	rBuf := bytes.NewReader(message[0:2])
	err := binary.Read(rBuf, binary.LittleEndian, &value)

	if err != nil {
		fmt.Println("binary.Read failed: ", err)
	} else {
		//fmt.Printf("from byte buffer, %s: %d\n", s, value)
	}

	return value
}
