package afp

import (
	"io"
	"fmt"
	"errors"
	"aaronlindsay.com/go/pkg/pso2/util"
	"github.com/quarnster/util/encoding/binary"
)

const (
	ModelHeaderMagic uint32 = 0x46425456 // little endian "VTBF"
)

type Model struct {
	reader io.ReadSeeker

	Header ModelHeader
	Entries []ModelEntry
}

func NewModel(reader io.ReadSeeker) (*Model, error) {
	m := &Model{ reader: reader }
	return m, m.parse()
}

type ModelHeader struct {
	Magic, HeaderSize uint32
	Type string `length:"4"`
	Unk uint32
}

type ModelEntry struct {
	Type string `length:"4"`
	Size uint32

	SubType string `length:"4"`

	Alignment []uint8 `skip:"Size-4" length:"0"`

	Data io.ReadSeeker `if:"0"`
}

func (h *ModelHeader) Validate() error {
	if h.Magic != ModelHeaderMagic {
		return errors.New("not a VTBF file")
	}

	if h.HeaderSize != 0x10 {
		return errors.New("header format error (size != 0x10)")
	}

	if h.Unk != 0x4c000001 {
		return errors.New("header format error (unk != 0x4c000001)")
	}

	return nil
}

func (m *Model) parse() (err error) {
	reader := binary.BinaryReader{ Reader: m.reader, Endianess: binary.LittleEndian }

	if err = reader.ReadInterface(&m.Header); err != nil {
		return
	}

	reader.Seek(int64(m.Header.HeaderSize) - 0x10, 1)

	offset := int64(m.Header.HeaderSize)
	for err == nil {
		entry := ModelEntry{}

		if err = reader.ReadInterface(&entry); err != nil {
			if err == io.EOF {
				return nil
			}
			return
		}

		entry.Data = io.NewSectionReader(util.ReaderAt(m.reader), offset + 0x0c, int64(entry.Size) - 0x04)
		offset += 0x08 + int64(entry.Size)

		m.Entries = append(m.Entries, entry)

		switch entry.SubType {
			case "NODE": // Bone data

			case "NODO": // More bone things

			case "VSET": // Vertex data shit
				err = parseModelEntryVSET(&entry)
		}
	}

	return
}

func (m *Model) Write(writer io.Writer) error {
	return nil
}

func parseModelEntryVSET(entry *ModelEntry) error {
	reader := binary.BinaryReader{ Reader: entry.Data, Endianess: binary.LittleEndian }

	var err error

	unk, err := reader.Uint16()
	count, err := reader.Uint16()

	fmt.Printf("Count: %04x\n", count)
	fmt.Printf("Unk: %04x\n", unk)

	for i := uint16(0); i < count && err == nil; i++ {
		var identifier uint16
		identifier, err = reader.Uint16()
		var data []uint8

		if err != nil {
			break
		}

		if identifier & 0x8000 != 0 { // Size is fucked up
			var sz, unk uint8
			unk, err = reader.Uint8()

			var size uint16
			if unk == 0x10 {
				size, err = reader.Uint16()
			} else if unk == 0x08 {
				sz, err = reader.Uint8();
				size = uint16(sz)
			} else {
				return errors.New("Unknown size flag")
			}

			fmt.Printf("Oh %02x, %04x\n", unk, size)
			x := 2
			data = make([]uint8, x * int(size + 1))
		} else if identifier & 0x0900 != 0 { // and 0x0900
			data = make([]uint8, 4)
		} else if identifier & 0x0600 == 0x0600 { // and 0x0600
			data = make([]uint8, 2)
		} else if identifier & 0xff00 != 0 {
			fmt.Printf("what the fuck is %04x\n", identifier & 0xff00)
			//break
		}

		_, err = util.ReadN(entry.Data, data)

		fmt.Printf("\t\t%04x (%08x): %x\n", identifier, len(data), data)
	}

	if _, err = entry.Data.Read(make([]uint8, 1)); err != io.EOF {
		return errors.New("Expected EOF")
	}

	if err == io.EOF {
		return nil
	}

	return err
}
