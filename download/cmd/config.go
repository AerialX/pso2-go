package cmd

import (
	"io"
	"bufio"
	"strings"
)

type TranslationConfig map[string]string

const (
	ConfigTranslationPath = "TranslationPath"
	ConfigPublicKeyPath = "PublicKeyPath"
	ConfigPublicKeyDump = "PublicKeyDump"
)

func NewTranslationConfig(reader io.Reader) (t TranslationConfig, err error) {
	t = make(TranslationConfig)

	r := bufio.NewReader(reader)
	for err != io.EOF {
		var line string
		line, err = r.ReadString('\n')
		line = strings.Replace(line, "\r", "", -1)
		s := strings.SplitN(line, ":", 2)
		if len(s) == 2 {
			t[s[0]] = s[1]
		}
	}

	return
}

func (t TranslationConfig) Write(w io.Writer) (err error) {
	for k, v := range t {
		line := k + ":" + v + "\n"

		_, err = w.Write([]byte(line))
		if err != nil {
			return
		}
	}

	return
}
