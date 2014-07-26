package util

import "io"

type readWriteWrapper struct {
	io.Reader
	io.Writer
}

func (r readWriteWrapper) Close() (err error) {
	if c, ok := r.Reader.(io.Closer); ok {
		err = c.Close()
	}

	if c, ok := r.Writer.(io.Closer); ok {
		e := c.Close()

		if e != nil {
			err = e
		}
	}

	return
}

func ReadWriter(reader io.Reader, writer io.Writer) io.ReadWriteCloser {
	return readWriteWrapper{reader, writer}
}
