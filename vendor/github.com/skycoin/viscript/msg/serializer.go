package msg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func init() {
	//msg.Serialize(0x0051, event)

	var m1 MessageMousePos
	m1.X = 0.15
	m1.Y = 72343

	x1 := Serialize(0x0051, m1)

	var m2 MessageMousePos
	MustDeserialize(x1, &m2)

	x2 := Serialize(0x0051, m2)

	for i := range x1 {
		if x1[i] != x2[i] {
			log.Panicf("serialization test failed: \n %x, \n %x \n", x1, x2)
		}
	}
}

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

/*
	TODO?

	//simplify with direct
	var value2 uint16
	value2 = uint16(message[0] << 8)
	value2 = value2 | uint16(message[1])

	if value != value2 {
		fmt.Printf("value1, value2= 0x%.4X, 0x%.4X \n", value, value2)

		value3 := uint16(message[0] << 8)
		value4 := uint16(message[1])

		fmt.Printf("value3, value4= 0x%.4X, 0x%.4X \n", value3, value4)
*/
