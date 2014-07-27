package packets

import (
	"net"
)

const TypeShip	= 0x3d11
const ShipCount = 10
var ShipHostnames [ShipCount]string = [ShipCount]string{
	"gs136.pso2gs.net",
	"gs001.pso2gs.net",
	"gs016.pso2gs.net",
	"gs031.pso2gs.net",
	"gs046.pso2gs.net",
	"gs061.pso2gs.net",
	"gs076.pso2gs.net",
	"gs091.pso2gs.net",
	"gs106.pso2gs.net",
	"gs121.pso2gs.net",
}

type ShipEntry struct {
	Unk1, Number uint32
	NameRaw [0x10]uint16
	AddressRaw [4]uint8
	Zero, Unk2 uint32
}

type Ship struct {
	Entries [ShipCount]ShipEntry
	Unk [3]uint32
}

func (s *ShipEntry) Name() string {
	return DecodeString(s.NameRaw[:])
}

func (s *ShipEntry) SetName(v string) {
	EncodeString(v, s.NameRaw[:])
}

func (s *ShipEntry) Address() net.IP {
	return net.IP(s.AddressRaw[:])
}

func (s *ShipEntry) SetAddress(v net.IP) {
	copy(s.AddressRaw[:], v.To4())
}

func (s *Ship) Packet() (*Packet, error) {
	return PacketFromBinary(TypeShip, FlagProcessed, s)
}

func ParseShip(p *Packet) (*Ship, error) {
	s, err := PacketToBinary(p, &Ship{})
	return s.(*Ship), err
}
