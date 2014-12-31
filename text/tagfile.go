package text

// NOTE: This file is all wrong, by the way. Only TagRead() works, the rest can be thrown out

import (
	"io"
	"aaronlindsay.com/go/pkg/pso2/util"
	"github.com/quarnster/util/encoding/binary"
	bin "encoding/binary"
)

type TagFileEntry struct {
	Tag string `length:"4"`
	Size uint32

	Data io.ReadSeeker `if:"0"`
}

type TagFile struct {
	Entries []TagFileEntry

	reader io.ReadSeeker
}

func NewTagFile(reader io.ReadSeeker) (*TagFile, error) {
	f := &TagFile{ nil, reader }
	return f, f.parse()
}

func (f *TagFile) parse() error {
	var err error
	reader := binary.BinaryReader{ Reader: f.reader, Endianess: binary.LittleEndian };

	offset := int64(0)

	var entry TagFileEntry
	for err = reader.ReadInterface(&entry); err == nil; err = reader.ReadInterface(&entry) {
		offset += 8
		entry.Data = io.NewSectionReader(util.ReaderAt(f.reader), offset, int64(entry.Size))

		f.Entries = append(f.Entries, entry)

		_, err = f.reader.Seek(int64(entry.Size), 1)
		offset += int64(entry.Size)
	}

	if err == io.EOF {
		return nil
	}

	return err
}

func TagRead(r io.ReadSeeker) (TagFileEntry, error) {
	reader := binary.BinaryReader{ Reader: r, Endianess: binary.LittleEndian };

	var err error
	var entry TagFileEntry
	if err = reader.ReadInterface(&entry); err == nil {
		offset, err := r.Seek(0, 1)
		entry.Data = io.NewSectionReader(util.ReaderAt(r), offset, int64(entry.Size))

		return entry, err
	}

	return TagFileEntry{}, err
}

func (f *TagFile) Write(writer io.Writer) (err error) {
	var n int64
	end := bin.LittleEndian

	for _, entry := range f.Entries {
		size := (entry.Size + 7) / 8 * 8

		bin.Write(writer, end, entry.Tag)
		bin.Write(writer, end, size)

		n, err = io.CopyN(writer, entry.Data, int64(entry.Size))

		if err == nil || err == io.EOF {
			if n < int64(size) {
				err = bin.Write(writer, end, make([]uint8, int64(size) - n))
			}
		}

		if err != nil {
			return
		}
	}

	return
}
