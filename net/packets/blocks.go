package packets

import (
	"bytes"
	"encoding/binary"
)

const TypeBlocks = 0x6511
const TypeBlocks2 = 0x1011

type Blocks struct {
	Count uint32
	Entries []Block `length:"Count"`
}

func (b *Blocks) Packet() (*Packet, error) {
	var buffer bytes.Buffer
	binary.Write(&buffer, binary.LittleEndian, uint32(len(b.Entries)))
	binary.Write(&buffer, binary.LittleEndian, b.Entries)

	return &Packet{TypeBlocks, 0, buffer.Bytes()}, nil
}

func ParseBlocks(p *Packet) (*Blocks, error) {
	s, err := PacketToBinary(p, &Blocks{})
	return s.(*Blocks), err
}
