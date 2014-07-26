package packets

import (
	"net"
	"unicode/utf16"
)

const TypeBlock = 0x00002c11

type Block struct {
	Unk [7]uint32 // First is count maybe?
	NameRaw [0x20]uint16
	AddressRaw [4]uint8
	Port uint16
	Unk2 [0x26]uint8
}

func (b *Block) Name() string {
	return string(utf16.Decode(b.NameRaw[:]))
}

func (b *Block) SetName(v string) {
	raw := utf16.Encode([]rune(v))
	copy(b.NameRaw[:], raw)
	for i := len(raw); i < len(b.NameRaw); i++ {
		b.NameRaw[i] = 0
	}
}

func (b *Block) Address() net.IP {
	return net.IP(b.AddressRaw[:])
}

func (b *Block) SetAddress(v net.IP) {
	v = v.To4()
	copy(b.AddressRaw[:], v)
}

func (b *Block) Packet() (*Packet, error) {
	return PacketFromBinary(TypeBlock, b)
}

func ParseBlock(p *Packet) (*Block, error) {
	s, err := PacketToBinary(p, &Block{})
	return s.(*Block), err
}

func packetBlock(p *Packet) (interface{}, error) {
	return ParseBlock(p)
}
