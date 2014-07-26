package net

import (
	"io"
	"net"
	"fmt"
	"errors"
	"crypto/cipher"
	"crypto/rc4"
	"encoding/binary"
	"aaronlindsay.com/go/pkg/pso2/net/packets"
)

const maxPacketSize = 0x04000000

type Connection struct {
	stream io.ReadWriter

	icipher cipher.Stream
	ocipher cipher.Stream
}

func NewConnection(stream io.ReadWriter) *Connection {
	c := &Connection{}
	c.stream = stream
	return c
}

func (c *Connection) RemoteAddr() net.Addr {
	return c.stream.(net.Conn).RemoteAddr()
}

func (c *Connection) LocalAddr() net.Addr {
	return c.stream.(net.Conn).LocalAddr()
}

func (c *Connection) SetCipher(key []uint8) (err error) {
	if key == nil {
		c.icipher = nil
		c.ocipher = nil
	} else {
		c.icipher, err = rc4.NewCipher(key)
		c.ocipher, err = rc4.NewCipher(key)
	}

	return
}

func (c *Connection) ReadPacket() (*packets.Packet, error) {
	var headerData [8]uint8

	read := 0
	for read < len(headerData) {
		n, err := c.stream.Read(headerData[read:])
		read += n

		if err != nil {
			return nil, err
		}
	}

	if c.icipher != nil {
		c.icipher.XORKeyStream(headerData[:], headerData[:])
	}

	p, size := &packets.Packet{binary.LittleEndian.Uint16(headerData[4:6]), binary.LittleEndian.Uint16(headerData[6:]), nil}, binary.LittleEndian.Uint32(headerData[:4])

	if size < 8 || size > maxPacketSize {
		return nil, errors.New("invalid packet size")
	}

	data := make([]uint8, size - 8)
	p.Data = data

	for len(data) > 0 {
		n, err := c.stream.Read(data)

		if err != nil {
			return p, err
		}

		data = data[n:]
	}

	if c.icipher != nil {
		c.icipher.XORKeyStream(p.Data, p.Data)
	}

	Logger.Tracef("%s read %s", c, p)

	return p, nil
}

func (c *Connection) WritePacket(p *packets.Packet) error {
	Logger.Tracef("%s writing %s", c, p)

	data := make([]uint8, 8 + len(p.Data))
	binary.LittleEndian.PutUint32(data[:4], uint32(len(data)))
	binary.LittleEndian.PutUint16(data[4:6], p.Type)
	binary.LittleEndian.PutUint16(data[6:8], p.Flags)
	copy(data[8:], p.Data)

	if c.ocipher != nil {
		c.ocipher.XORKeyStream(data, data)
	}

	for len(data) > 0 {
		w, err := c.stream.Write(data)

		if err != nil {
			return err
		}

		data = data[w:]
	}

	return nil
}

func (c *Connection) RoutePackets(r *PacketRoute) error {
	for {
		p, err := c.ReadPacket()

		if err != nil {
			return err
		}

		consumed, err := r.RoutePacket(c, p)

		if err != nil {
			return err
		}

		if !consumed {
			Logger.Warningf("%s packet %s ignored", c, p)
		}
	}
}

func (c *Connection) Close() error {
	if s, ok := c.stream.(io.Closer); ok {
		return s.Close()
	}

	return nil
}

func (c *Connection) String() string {
	if c, ok := c.stream.(net.Conn); ok {
		return fmt.Sprintf("[pso2/net/connection: %s -> %s]", c.LocalAddr(), c.RemoteAddr())
	} else {
		return "[pso2/net/connection]"
	}
}
