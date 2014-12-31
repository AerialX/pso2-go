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
	entries []archiveEntry `if:"0"`
}

func NewArchive(reader io.ReadSeeker) (*Archive, error) {
	a := &Archive{ reader: reader }
	return a, a.parse()
}

type archiveHeader struct {
	Magic, EntryCount, Zero, Count2 uint32
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
		return errors.New("not an AFP archive")
	}

	if h.Zero != 0 {
		return errors.New("header format error (zero != 0)")
	}

	if h.Count2 != 1 {
		return errors.New("header format error (unk != 1)")
	}

	return nil
}

func (a *Archive) parse() (err error) {
	reader := binary.BinaryReader{ Reader: a.reader, Endianess: binary.LittleEndian }

	if err = reader.ReadInterface(&a.header); err != nil {
		return
	}

	entryOffset := int64(0x10)
	a.entries = make([]archiveEntry, a.header.EntryCount)
	for i := uint32(0); i < a.header.EntryCount; i++ {
		entry := &a.entries[i]

		if err = reader.ReadInterface(entry); err != nil {
			return
		}

		entry.Data = io.NewSectionReader(util.ReaderAt(a.reader), entryOffset + int64(entry.DataOffset), int64(entry.DataSize))
		entryOffset += int64(entry.DataEnd)
	}

	return
}

func (a *Archive) Write(writer io.Writer) error {
	return nil
}

type Entry struct {
	Type, Name string
	Size uint32
	Data io.ReadSeeker

	file *archiveEntry
}

func (a *Archive) EntryCount() int {
	return len(a.entries)
}

func (a *Archive) Entry(i int) Entry {
	entry := &a.entries[i]

	return Entry{entry.Type, entry.Name, entry.DataSize, entry.Data, entry}
}
