package util

import (
	"io"
	"os"
)

func CopyFile(source, dest string) (err error) {
	df, err := os.Create(dest)
	if err != nil {
		return
	}

	sf, err := os.Open(source)
	if err != nil {
		df.Close()
		return
	}

	_, err = io.Copy(df, sf)

	sf.Close()
	df.Close()

	return
}
