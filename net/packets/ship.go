package packets

import (
	"net"
	"unicode/utf16"
)

const ShipCount = 10
const TypeShip	= 0x00043d11
var ShipHostnames [ShipCount]string = [ShipCount]string{
	"gs001.pso2gs.net",
	"gs016.pso2gs.net",
	"gs031.pso2gs.net",
	"gs046.pso2gs.net",
	"gs061.pso2gs.net",
	"gs076.pso2gs.net",
	"gs091.pso2gs.net",
	"gs106.pso2gs.net",
	"gs121.pso2gs.net",
	"gs136.pso2gs.net",
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
	return string(utf16.Decode(s.NameRaw[:]))
}

func (s *ShipEntry) SetName(v string) {
	raw := utf16.Encode([]rune(v))
	copy(s.NameRaw[:], raw)
	for i := len(raw); i < len(s.NameRaw); i++ {
		s.NameRaw[i] = 0
	}
}

func (s *ShipEntry) Address() net.IP {
	return net.IP(s.AddressRaw[:])
}

func (s *ShipEntry) SetAddress(v net.IP) {
	v = v.To4()
	copy(s.AddressRaw[:], v)
}

func (s *Ship) Packet() (*Packet, error) {
	return PacketFromBinary(TypeShip, s)
}

func ParseShip(p *Packet) (*Ship, error) {
	s, err := PacketToBinary(p, &Ship{})
	return s.(*Ship), err
}

func packetShip(p *Packet) (interface{}, error) {
	return ParseShip(p)
}
