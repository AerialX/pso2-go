package util

import (
	"io"
	"bufio"
)

type bufReader struct {
	reader io.ReadSeeker
	breader *bufio.Reader
	position int64
}

func (b *bufReader) Read(p []uint8) (n int, err error) {
	n, err = b.breader.Read(p)

	b.position += int64(n)

	return
}

func (b *bufReader) Seek(offset int64, whence int) (n int64, err error) {
	relative := int64(-1)
	if whence == 1 {
		relative = offset
	} else if whence == 0 {
		relative = offset - b.position
	}

	if relative == 0 {
		n = b.position
	} else if relative >= 0 && relative < int64(b.breader.Buffered()) {
		var buffer [0x100]uint8
		for relative > 0  && err == nil {
			diff := relative
			if diff > int64(len(buffer)) {
				diff = int64(len(buffer))
			}

			var r int
			r, err = b.breader.Read(buffer[:diff])
			b.position += int64(r)
			relative -= int64(r)
		}

		n = b.position
	} else {
		if whence == 1 {
			b.position, err = b.reader.Seek(b.position + offset, 0)
		} else {
			n, err = b.reader.Seek(offset, whence)
			b.position = n
		}

		b.breader.Reset(b.reader)
	}

	return
}

func BufReader(reader io.ReadSeeker) io.ReadSeeker {
	return &bufReader{reader, bufio.NewReader(reader), 0}
}

func BufReaderSize(reader io.ReadSeeker, size int) io.ReadSeeker {
	return &bufReader{reader, bufio.NewReaderSize(reader, size), 0}
}
