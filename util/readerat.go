package util

import (
	"io"
)

type readerAtWrapper struct {
	io.ReadSeeker
}

func (r readerAtWrapper) ReadAt(p []byte, off int64) (int, error) {
	r.Seek(off, 0)
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
