package packets

import (
	"errors"
	"crypto/rsa"
	"crypto/rand"
)

const TypeCipher	= 0x00000b11

type Cipher struct {
	KeyData [0x80]uint8
	Padding [0x84]uint8
}

func (c *Cipher) Key(key *rsa.PrivateKey) ([]uint8, error) {
	data := make([]uint8, len(c.KeyData))
	for i := range data {
		data[i] = c.KeyData[len(c.KeyData) - i - 1]
	}

	return rsa.DecryptPKCS1v15(nil, key, data)
}

func (c *Cipher) RC4Key(key *rsa.PrivateKey) ([]uint8, error) {
	data, err := c.Key(key)
	if err != nil {
		return nil, err
	}

	return CipherRC4Key(data)
}

func CipherRC4Key(key []uint8) ([]uint8, error) {
	if len(key) != 0x20 {
		return nil, errors.New("pso2/net/packets/Cipher: invalid decrypted key length")
	}

	return key[0x10:], nil
}

func (c *Cipher) SetKey(v []uint8, key *rsa.PublicKey) error {
	data, err := rsa.EncryptPKCS1v15(rand.Reader, key, v)

	if err != nil {
		return err
	}

	if len(data) != len(c.KeyData) {
		return errors.New("pso2/net/packets/Cipher: invalid encrypted key length")
	}

	for i := range c.KeyData {
		c.KeyData[i] = data[len(c.KeyData) - i - 1]
	}

	return nil
}

func (s *Cipher) Packet() (*Packet, error) {
	return PacketFromBinary(TypeCipher, s)
}

func ParseCipher(p *Packet) (*Cipher, error) {
	s, err := PacketToBinary(p, &Cipher{})
	return s.(*Cipher), err
}

func packetCipher(p *Packet) (interface{}, error) {
	return ParseCipher(p)
}
