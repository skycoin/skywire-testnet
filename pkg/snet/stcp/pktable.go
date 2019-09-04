package stcp

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/skycoin/dmsg/cipher"
	"os"
	"path/filepath"
	"strings"
)

type PKTable interface {
	Addr(pk cipher.PubKey) (string, bool)
	PubKey(addr string) (cipher.PubKey, bool)
	Count() int
}

type memoryTable struct {
	entries map[cipher.PubKey]string
	reverse map[string]cipher.PubKey
}

func NewTable(entries map[cipher.PubKey]string) PKTable {
	reverse := make(map[string]cipher.PubKey, len(entries))
	for pk, addr := range entries {
		reverse[addr] = pk
	}
	return &memoryTable{
		entries: entries,
		reverse: reverse,
	}
}

func NewTableFromFile(path string) (PKTable, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println("tcp_factory: failed to close table file:", err)
		}
	}()

	var (
		entries = make(map[cipher.PubKey]string)
		s       = bufio.NewScanner(f)
	)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) != 2 {
			return nil, errors.New("pk file is invalid: each line should have two fields")
		}
		var pk cipher.PubKey
		if err := pk.UnmarshalText([]byte(fields[0])); err != nil {
			return nil, fmt.Errorf("pk file is invalid: each line should have two fields: %v", err)
		}
		entries[pk] = fields[1]
	}
	return NewTable(entries), nil
}

func (mt *memoryTable) Addr(pk cipher.PubKey) (string, bool) {
	addr, ok := mt.entries[pk]
	return addr, ok
}

func (mt *memoryTable) PubKey(addr string) (cipher.PubKey, bool) {
	pk, ok := mt.reverse[addr]
	return pk, ok
}

func (mt *memoryTable) Count() int {
	return len(mt.entries)
}
