package conn

import (
	"errors"
	"github.com/klauspost/reedsolomon"
	"github.com/skycoin/net/util"
)

type fecDecoder struct {
	dataShards   int
	parityShards int
	shardSize    uint32

	lowestGroup uint32
	groups      map[uint32]*group

	codec reedsolomon.Encoder
}

type group struct {
	datas     [][]byte
	dataRecv  []bool
	dataCount int
	count     int
	startSeq  uint32
	recovered bool
	maxSize   int
}

func newFECDecoder(dataShards, parityShards int) *fecDecoder {
	fec := &fecDecoder{
		dataShards:   dataShards,
		parityShards: parityShards,
		shardSize:    uint32(dataShards + parityShards),

		groups: make(map[uint32]*group),
	}

	var err error
	fec.codec, err = reedsolomon.New(dataShards, parityShards, reedsolomon.WithMaxGoroutines(1))
	if err != nil {
		return nil
	}
	return fec
}

func (fec *fecDecoder) decode(seq uint32, data []byte) (g *group, err error) {
	sz := len(data)
	if sz <= 0 {
		err = errors.New("empty fec data")
		return
	}
	seq = seq - 1
	gindex := seq / fec.shardSize
	if gindex < fec.lowestGroup {
		return
	}

	g, ok := fec.groups[gindex]
	if !ok {
		g = &group{
			startSeq: gindex * fec.shardSize,
			datas:    make([][]byte, fec.shardSize),
			dataRecv: make([]bool, fec.dataShards),
		}
		fec.groups[gindex] = g
	}
	if g == nil {
		return
	}

	index := seq % fec.shardSize
	if g.datas[index] == nil {
		if sz > g.maxSize {
			g.maxSize = sz
		}
		g.count++
		if index < uint32(fec.dataShards) {
			g.dataCount++
			g.dataRecv[index] = true
		}
		g.datas[index] = util.FixedMtuPool.Get()
		g.datas[index] = g.datas[index][:sz]
		copy(g.datas[index], data)
	} else {
		return nil, nil
	}

	if g.dataCount == fec.dataShards {
		goto OK
	}

	if g.count >= fec.dataShards {
		cache := g.datas
		for k, v := range cache {
			if v == nil {
				continue
			}
			s := len(v)
			util.XorBytes(v[s:g.maxSize], v[s:g.maxSize], v[s:g.maxSize])
			cache[k] = v[:g.maxSize]
		}
		if err = fec.codec.ReconstructData(cache); err == nil {
			g.recovered = true
			goto OK
		}
	}

	return nil, err
OK:
	if fec.lowestGroup == gindex {
		for {
			fec.lowestGroup++
			group := fec.groups[gindex]
			if group != nil {
				for _, v := range group.datas {
					if len(v) > 0 {
						util.FixedMtuPool.Put(v)
					}
				}
			}

			delete(fec.groups, gindex)
			if len(fec.groups) < 1 {
				break
			}
			gindex++
			g, ok := fec.groups[gindex]
			if !ok || g != nil {
				break
			}
		}
	} else {
		for _, v := range fec.groups[gindex].datas {
			if len(v) > 0 {
				util.FixedMtuPool.Put(v)
			}
		}
		fec.groups[gindex] = nil
	}

	return
}

type fecEncoder struct {
	dataShards   int
	parityShards int
	shardSize    uint32

	count   int
	maxSize int

	cache    [][]byte
	tmpCache [][]byte

	codec reedsolomon.Encoder
}

func newFECEncoder(dataShards, parityShards int) *fecEncoder {
	fec := &fecEncoder{
		dataShards:   dataShards,
		parityShards: parityShards,
		shardSize:    uint32(dataShards + parityShards),
	}

	var err error
	fec.codec, err = reedsolomon.New(dataShards, parityShards, reedsolomon.WithMaxGoroutines(1))
	if err != nil {
		return nil
	}

	fec.cache = make([][]byte, fec.shardSize)
	fec.tmpCache = make([][]byte, fec.shardSize)
	for k := range fec.cache {
		fec.cache[k] = make([]byte, 1500)
	}
	return fec
}

func (fec *fecEncoder) encode(data []byte) (datas [][]byte, err error) {
	sz := len(data)
	fec.cache[fec.count] = fec.cache[fec.count][:sz]
	copy(fec.cache[fec.count], data)
	if sz > fec.maxSize {
		fec.maxSize = sz
	}
	fec.count++

	if fec.count == fec.dataShards {
		for i := 0; i < fec.dataShards; i++ {
			shard := fec.cache[i]
			s := len(shard)
			util.XorBytes(shard[s:fec.maxSize], shard[s:fec.maxSize], shard[s:fec.maxSize])
		}

		c := fec.tmpCache
		for k, v := range fec.cache {
			c[k] = v[:fec.maxSize]
		}

		if err = fec.codec.Encode(c); err == nil {
			datas = fec.cache[fec.dataShards:]
			for i, v := range datas {
				datas[i] = v[:fec.maxSize]
			}
		}
		fec.maxSize = 0
		fec.count = 0
	}

	return
}
