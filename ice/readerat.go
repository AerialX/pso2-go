package ice

import (
	"io"
	"errors"
)

type readerAtWrapper struct {
	io.ReadSeeker
}

func (r readerAtWrapper) ReadAt(p []byte, off int64) (int, error) {
	r.Seek(off, 0)
	return r.Read(p)
}

func readerAt(reader io.Reader) io.ReaderAt {
	switch r := reader.(type) {
		case io.ReaderAt:
			return r
		case io.ReadSeeker:
			return readerAtWrapper{r}
		default:
			return readerAtWrapper{seeker(reader)}
	}

	panic("readerAt")
}

type closeGuardWrapper struct {
	io.Writer
}

func (c closeGuardWrapper) Close() error {
	return nil
}

func closeGuard(writer io.Writer) io.Writer {
	return closeGuardWrapper{writer}
}

type seekerWrapper struct {
	io.Reader
	position int64
}

func (s seekerWrapper) Seek(offset int64, whence int) (int64, error) {
	// Determine offset...
	switch whence {
		case 1:
			offset += s.position
		case 2:
			return s.position, errors.New("unsupported seek")
	}

	if offset < s.position {
		return s.position, errors.New("unsupported seek")
	}

	// Read forward until we reach our destination
	var err error
	for offset > s.position && err == nil {
		diff := offset - s.position
		var buffer [0x400]uint8

		if diff > int64(len(buffer)) {
			diff = int64(len(buffer))
		}

		var read int
		read, err = s.Read(buffer[:diff])
		s.position += int64(read)
	}

	return s.position, err
}

func seeker(reader io.Reader) io.ReadSeeker {
	switch r := reader.(type) {
		case io.ReadSeeker:
			return r
		default:
			return seekerWrapper{reader, 0}
	}
}
