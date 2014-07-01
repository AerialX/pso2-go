package util

import (
	"io"
)

type closeGuardWrapper struct {
	io.Writer
}

func (c closeGuardWrapper) Close() error {
	return nil
}

func CloseGuard(writer io.Writer) io.Writer {
	return closeGuardWrapper{writer}
}
