package packets

import (
	"net"
)

const TypeBlockResponse = 0x1311

type BlockResponse struct {
	Unk [0x0c]uint8
	AddressRaw [4]uint8
	Port uint16
	Unk2 [0x0a]uint8
}

func (b *BlockResponse) Address() net.IP {
	return net.IP(b.AddressRaw[:])
}

func (b *BlockResponse) SetAddress(v net.IP) {
	copy(b.AddressRaw[:], v.To4())
}

func (b *BlockResponse) Packet() (*Packet, error) {
	return PacketFromBinary(TypeBlockResponse, 0, b)
}

func ParseBlockResponse(p *Packet) (*BlockResponse, error) {
	s, err := PacketToBinary(p, &BlockResponse{})
	return s.(*BlockResponse), err
}
