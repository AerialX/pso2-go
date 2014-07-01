package afp

import (
	"io"
	"errors"
	"aaronlindsay.com/go/pkg/pso2/util"
	"github.com/quarnster/util/encoding/binary"
)

const (
	HeaderMagic uint32 = 0x00706661 // little endian "afp\0"
)

type Archive struct {
	reader io.ReadSeeker
	header archiveHeader
}

func NewArchive(reader io.ReadSeeker) (*Archive, error) {
	a := &Archive{ reader: reader }
	return a, a.parse()
}

type archiveHeader struct {
	Magic, EntryCount, Zero, Count2 uint32

	Entries []archiveEntry `length:"EntryCount"`
}

type archiveEntry struct {
	Name string `length:"0x20"`
	DataSize, DataOffset, DataEnd uint32
	Type string `length:"4"`

	Alignment []uint8 `skip:"DataEnd-0x30" length:"0"`

	Data io.ReadSeeker `if:"0"`
}

func (h *archiveHeader) Validate() error {
	if h.Magic != HeaderMagic {
		return errors.New("not a AFP archive")
	}

	if h.Zero != 0 {
		return errors.New("zero != 0")
	}

	if h.Count2 != 1 {
		return errors.New("unk != 1")
	}

	return nil
}

func (a *Archive) parse() (err error) {
	reader := binary.BinaryReader{a.reader, binary.LittleEndian}

	if err = reader.ReadInterface(&a.header); err != nil {
		return err
	}

	entryOffset := int64(0)
	for i, entry := range a.header.Entries {
		a.header.Entries[i].Data = io.NewSectionReader(util.ReaderAt(a.reader), entryOffset + int64(entry.DataOffset), int64(entry.DataSize))
		entryOffset += int64(entry.DataEnd)
	}

	return
}

func (a *Archive) Write(writer io.Writer) error {
	return nil
}
