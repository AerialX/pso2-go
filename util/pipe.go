package util

import "io"

type PipeWriterType interface {
	Write(writer io.Writer) error
}

func PipeReader(writer PipeWriterType) *io.PipeReader {
	r, w := io.Pipe()
	go func() {
		w.CloseWithError(writer.Write(w))
	}()
	return r
}
