package net

import (
	"io"
	"io/ioutil"
	"errors"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

func loadKey(reader io.Reader) (interface{}, error) {
	data, err := ioutil.ReadAll(reader)

	if err != nil {
		return nil, err
	}

	for len(data) > 0 {
		var block *pem.Block
		block, data = pem.Decode(data)

		if block == nil {
			break
		}

		switch block.Type {
			case "RSA PRIVATE KEY":
				return x509.ParsePKCS1PrivateKey(block.Bytes)

			case "PRIVATE KEY":
				return x509.ParsePKCS8PrivateKey(block.Bytes)

			case "RSA PUBLIC KEY": fallthrough
			case "PUBLIC KEY":
				return x509.ParsePKIXPublicKey(block.Bytes)
		}
	}

	return nil, errors.New("no key found")
}

func LoadPublicKey(reader io.Reader) (*rsa.PublicKey, error) {
	key, err := loadKey(reader)

	if err != nil {
		return nil, err
	}

	switch k := key.(type) {
		case *rsa.PublicKey:
			return k, nil
		case *rsa.PrivateKey:
			return &k.PublicKey, nil
	}

	return nil, errors.New("public key not found")
}

func LoadPrivateKey(reader io.Reader) (*rsa.PrivateKey, error) {
	key, err := loadKey(reader)

	if err != nil {
		return nil, err
	}

	if k, ok := key.(*rsa.PrivateKey); ok {
		return k, nil
	}

	return nil, errors.New("private key not found")
}
