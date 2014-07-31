package util

import (
	"io"
	"errors"
)

type readerAtWrapper struct {
	io.ReadSeeker
}

func (r readerAtWrapper) ReadAt(p []byte, off int64) (int, error) {
	n, err := r.Seek(off, 0)
	if err == nil && n != off {
		err = errors.New("unable to seek to requested offset")
	}

	if err != nil {
		return 0, err
	}

	return r.Read(p)
}

func ReaderAt(reader io.Reader) io.ReaderAt {
	switch r := reader.(type) {
		case io.ReaderAt:
			return r
		case io.ReadSeeker:
			return readerAtWrapper{r}
		default:
			return readerAtWrapper{Seeker(reader)}
	}

	panic("readerAt")
}
