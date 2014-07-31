package util

import (
	"io"
	"errors"
)

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
	var buffer [0x400]uint8
	for offset > s.position && err == nil {
		diff := offset - s.position

		if diff > int64(len(buffer)) {
			diff = int64(len(buffer))
		}

		var read int
		read, err = s.Read(buffer[:diff])
		s.position += int64(read)
	}

	return s.position, err
}

func Seeker(reader io.Reader) io.ReadSeeker {
	switch r := reader.(type) {
		case io.ReadSeeker:
			return r
		default:
			return seekerWrapper{reader, 0}
	}
}
