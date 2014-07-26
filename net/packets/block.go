package packets

import (
	"net"
)

const TypeBlock = 0x2c11

type Block struct {
	Unk [7]uint32
	NameRaw [0x20]uint16
	AddressRaw [4]uint8
	Port uint16
	Unk2 [0x26]uint8
}

func (b *Block) Name() string {
	return DecodeString(b.NameRaw[:])
}

func (b *Block) SetName(v string) {
	EncodeString(v, b.NameRaw[:])
}

func (b *Block) Address() net.IP {
	return net.IP(b.AddressRaw[:])
}

func (b *Block) SetAddress(v net.IP) {
	copy(b.AddressRaw[:], v.To4())
}

func (b *Block) Packet() (*Packet, error) {
	return PacketFromBinary(TypeBlock, 0, b)
}

func ParseBlock(p *Packet) (*Block, error) {
	s, err := PacketToBinary(p, &Block{})
	return s.(*Block), err
}
