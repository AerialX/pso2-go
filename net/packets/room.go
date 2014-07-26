package packets

import (
	"net"
)

const TypeRoom = 0x1711
const TypeRoomTeam = 0x4f11

type Room struct {
	Unk1 [6]uint32
	AddressRaw [4]uint8
	Unk2 uint32
	Port uint16
	Unk3 [6]uint8
}

func (b *Room) Address() net.IP {
	return net.IP(b.AddressRaw[:])
}

func (b *Room) SetAddress(v net.IP) {
	copy(b.AddressRaw[:], v.To4())
}

func (b *Room) Packet() (*Packet, error) {
	return PacketFromBinary(TypeRoom, 0, b)
}

func ParseRoom(p *Packet) (*Room, error) {
	s, err := PacketToBinary(p, &Room{})
	return s.(*Room), err
}

type RoomTeam Room

func (b *RoomTeam) Packet() (*Packet, error) {
	return PacketFromBinary(TypeRoomTeam, 0, b)
}
