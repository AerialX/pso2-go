package text

import (
	"io"
	"errors"
	"aaronlindsay.com/go/pkg/pso2/util"
	"github.com/quarnster/util/encoding/binary"
	"unicode/utf8"
	bin "encoding/binary"
)

const textBufferThreshold = 0x80000
const textBufferDataThreshold = 0x800000

type TextPair struct {
	Identifier, String string

	identifierIndex, stringIndex int
}

type TextFile struct {
	Entries []TextEntry

	Pairs []TextPair

	hasNEND bool
}

type TextEntry struct {
	Value []uint32
	Text string
	TextStatus int
}

const (
	TextEntryNone = iota
	TextEntryString
	TextEntryIdentifier
)

func NewTextFile(reader io.ReadSeeker) (*TextFile, error) {
	f := &TextFile{nil, nil, true}
	return f, f.parse(reader)
}

func (t *TextFile) parse(r io.ReadSeeker) (err error) {
	nifl, err := TagRead(r)

	if err != nil {
		return err
	}

	if nifl.Tag != "NIFL" {
		return errors.New("NIFL tag expected")
	}

	reader := binary.BinaryReader{ nifl.Data, binary.LittleEndian }

	type niflHeaderType struct {
		Unk, OffsetREL0, SizeREL0, OffsetNOF0, SizeNOF0 uint32
	}

	var niflHeader niflHeaderType
	if err = reader.ReadInterface(&niflHeader); err != nil {
		return err
	}

	if niflHeader.Unk != 1 {
		return errors.New("NIFL header magic != 1")
	}

	r.Seek(int64(niflHeader.OffsetREL0), 0)

	rel0, err := TagRead(r)
	if rel0.Tag != "REL0" {
		return errors.New("REL0 tag expected")
	}

	r.Seek(int64(niflHeader.OffsetNOF0), 0)

	nof0, err := TagRead(r)
	if nof0.Tag != "NOF0" {
		return errors.New("NOF0 tag expected")
	}

	nof0reader := binary.BinaryReader{ nof0.Data, binary.LittleEndian }
	count, err := nof0reader.Uint32()
	offsets := make([]uint32, int(count) + 1)
	i := 0
	for offset, _ := nof0reader.Uint32(); i < int(count); i++ {
		end, _ := nof0reader.Uint32()

		offsets[i] = end - offset

		offset = end

		if offsets[i] % 4 != 0 {
			return errors.New("nof0 entry not a multiple of 32 bits")
		}
	}
	offsets[i] = 8

	_, err = r.Seek(int64((nof0.Size + 8 + 0x0f) / 0x10 * 0x10) - 8, 1)

	nend, err := TagRead(r)
	if err == io.EOF || (nend.Tag == "" && nend.Size == 0) {
		t.hasNEND = false
	} else if nend.Tag != "NEND" {
		return errors.New("NEND tag expected")
	} else {
		t.hasNEND = true
	}

	t.Entries = make([]TextEntry, len(offsets))

	var rel0data io.ReadSeeker
	var rel0strings io.ReadSeeker
	reader = binary.BinaryReader{ rel0.Data, binary.LittleEndian }
	rel0size, err := reader.Uint32()
	rel0data = io.NewSectionReader(util.ReaderAt(rel0.Data), 8, int64(rel0size))
	rel0strings = io.NewSectionReader(util.ReaderAt(rel0.Data), int64(rel0size), int64(rel0.Size - rel0size))

	if rel0size < textBufferDataThreshold {
		rel0data, err = util.MemReader(rel0data)
		if err != nil {
			return err
		}
	}

	if rel0.Size - rel0size < textBufferThreshold {
		rel0strings, err = util.MemReader(rel0strings)
		if err != nil {
			return err
		}
	}

	rel0reader := binary.BinaryReader{ rel0data, binary.LittleEndian }

	pairMode := false
	var pair *string
	var pairi int
	for i, offset := range offsets {
		entry := &t.Entries[i]

		entry.Value = make([]uint32, offset / 4)
		for i := 0; i < int(offset / 4); i++ {
			entry.Value[i], err = rel0reader.Uint32()
		}

		if entry.Value[0] == 0xffffffff {
			pairMode = true
		} else if entry.Value[0] == 0x14 {
			pairMode = false
			pair = nil
		}

		if len(entry.Value) == 1 && entry.Value[0] != 0xffffffff {
			rel0strings.Seek(int64(entry.Value[0] - rel0size - 8), 0)
			charSize := 1
			if pair != nil {
				charSize = 2
			}
			entry.Text, _ = readString(charSize, rel0strings)

			if pair != nil {
				entry.TextStatus = TextEntryString
				t.Pairs = append(t.Pairs, TextPair{*pair, entry.Text, pairi, i})
				pair = nil
			} else {
				entry.TextStatus = TextEntryIdentifier
				if pairMode {
					pair = &entry.Text
					pairi = i
				}
			}
		}
	}

	return err
}

func readString(charSize int, reader io.Reader) (string, error) {
	stringValue := make([]rune, 0)
	for len(stringValue) == 0 || stringValue[len(stringValue) - 1] != 0 {
		p := make([]uint8, charSize)
		_, err := reader.Read(p)

		if err != nil {
			return string(stringValue), err
		}

		if charSize == 1 {
			stringValue = append(stringValue, rune(p[0]))
		} else {
			stringValue = append(stringValue, rune((uint16(p[1]) << 8) | uint16(p[0])))
		}
	}

	return string(stringValue[:len(stringValue) - 1]), nil
}

func (t *TextFile) Write(writer io.Writer) error {
	end := bin.LittleEndian

	// TODO: This reuse detection is pretty bad...
	stringReuse := make(map[string]TextEntry)

	entrySize := uint32(0)
	stringSize := uint32(0)
	for _, entry := range t.Entries {
		entrySize += uint32(len(entry.Value) * 4)

		if entry.TextStatus != TextEntryNone {
			if reuse, ok := stringReuse[entry.Text]; ok && reuse.TextStatus == entry.TextStatus {
				entry.Value[0] = reuse.Value[0]
			} else {
				entry.Value[0] = stringSize

				charSize := 1
				strlen := len(entry.Text)
				if entry.TextStatus == TextEntryString {
					charSize = 2
					strlen = utf8.RuneCountInString(entry.Text)
				}

				stringSize += uint32(charSize * (strlen + 1))
				stringSize = (stringSize + 3) / 4 * 4

				stringReuse[entry.Text] = entry
			}
		}
	}

	rel0size := entrySize + stringSize + 0x10
	rel0sizeRounded := (rel0size + 0x0f) / 0x10 * 0x10
	nof0size := uint32(len(t.Entries) + 1) * 4 + 8
	nof0sizeRounded := (nof0size + 0x0f) / 0x10 * 0x10

	io.WriteString(writer, "NIFL")
	bin.Write(writer, end, uint32(0x18))
	bin.Write(writer, end, uint32(0x01))
	bin.Write(writer, end, uint32(0x20))
	bin.Write(writer, end, rel0sizeRounded)
	bin.Write(writer, end, rel0sizeRounded + 0x20)
	bin.Write(writer, end, nof0sizeRounded)
	bin.Write(writer, end, uint32(0))

	io.WriteString(writer, "REL0")
	bin.Write(writer, end, rel0size - 8)
	bin.Write(writer, end, entrySize + 8)
	bin.Write(writer, end, uint32(0))

	for _, entry := range t.Entries {
		if entry.TextStatus != TextEntryNone {
			entry.Value[0] += entrySize + 0x10
		}

		bin.Write(writer, end, entry.Value)
	}

	stringReuse = make(map[string]TextEntry)

	for _, entry := range t.Entries {
		if reuse, ok := stringReuse[entry.Text]; ok && reuse.TextStatus == entry.TextStatus {
			continue
		}

		strlen := len(entry.Text) + 1
		switch entry.TextStatus {
			case TextEntryIdentifier:
				io.WriteString(writer, entry.Text)
				bin.Write(writer, end, uint8(0))
			case TextEntryString:
				for _, r := range entry.Text {
					bin.Write(writer, end, uint16(r))
				}
				bin.Write(writer, end, uint16(0))
				strlen = (utf8.RuneCountInString(entry.Text) + 1) * 2
			default:
				continue
		}

		padding := 4 - (strlen % 4)
		if padding < 4 {
			writer.Write(make([]uint8, padding))
		}

		stringReuse[entry.Text] = entry
	}

	writer.Write(make([]uint8, rel0sizeRounded - rel0size))

	io.WriteString(writer, "NOF0")
	bin.Write(writer, end, nof0size - 8)
	bin.Write(writer, end, uint32(len(t.Entries) - 1))

	offset := uint32(0x10)
	for _, entry := range t.Entries {
		bin.Write(writer, end, offset)
		offset += uint32(len(entry.Value) * 4)
	}

	writer.Write(make([]uint8, nof0sizeRounded - nof0size))

	if t.hasNEND {
		io.WriteString(writer, "NEND")
		bin.Write(writer, end, uint32(0x08))
		bin.Write(writer, end, uint32(0))
		bin.Write(writer, end, uint32(0))
	}

	return nil
}

func (t *TextFile) PairIdentifier(p *TextPair) *TextEntry {
	return &t.Entries[p.identifierIndex]
}

func (t *TextFile) PairString(p *TextPair) *TextEntry {
	return &t.Entries[p.stringIndex]
}
