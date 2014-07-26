package packets

import (
	"fmt"
	"bytes"
	"github.com/quarnster/util/encoding/binary"
	bin "encoding/binary"
	"aaronlindsay.com/go/pkg/pso2/util"
)

const FlagProcessed = 4

type Packet struct {
	Type uint16
	Flags uint16
	Data []uint8
}

func (p *Packet) String() string {
	return fmt.Sprintf("[pso2/net/packet: 0x%04x (0x%04x) size: 0x%08x]", p.Type, p.Flags, len(p.Data))
}

type PacketData interface {
	Packet() (*Packet, error)
}

func PacketToBinary(p *Packet, i interface{}) (interface{}, error) {
	reader := binary.BinaryReader{util.Seeker(bytes.NewBuffer(p.Data)), binary.LittleEndian}

	err := reader.ReadInterface(i)

	return i, err
}

func PacketFromBinary(packetType, flags uint16, i interface{}) (*Packet, error) {
	data := make([]uint8, 0, bin.Size(i))
	b := bytes.NewBuffer(data)
	err := bin.Write(b, bin.LittleEndian, i)

	if err != nil {
		return nil, err
	}

	return &Packet{packetType, flags, b.Bytes()}, nil
}
