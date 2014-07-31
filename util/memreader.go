package util

import (
	"io"
	"io/ioutil"
	"bytes"
)

func MemReader(reader io.Reader) (io.ReadSeeker, error) {
	buffer, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buffer), nil
}
