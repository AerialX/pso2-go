package util

import "io"

func ReadN(reader io.Reader, p []uint8) (n int, err error) {
	for len(p) > 0 && err == nil {
		var r int
		r, err = reader.Read(p)

		n += r
		p = p[r:]
	}

	return
}
