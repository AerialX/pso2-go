package net

import (
	"net"
	"fmt"
	"errors"
	"crypto/cipher"
	"crypto/rc4"
	"encoding/binary"
	"aaronlindsay.com/go/pkg/pso2/net/packets"
)

const maxPacketSize = 0x4000000

type Connection struct {
	conn net.Conn

	icipher cipher.Stream
	ocipher cipher.Stream
}

func NewConnection(conn net.Conn) *Connection {
	c := &Connection{}
	c.conn = conn
	return c
}

func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
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
		n, err := c.conn.Read(headerData[read:])
		read += n

		if err != nil {
			return nil, err
		}
	}

	if c.icipher != nil {
		c.icipher.XORKeyStream(headerData[:], headerData[:])
	}

	p, size := &packets.Packet{binary.LittleEndian.Uint32(headerData[4:]), nil}, binary.LittleEndian.Uint32(headerData[:4])

	if size < 8 || size > maxPacketSize {
		return nil, errors.New("invalid packet size")
	}

	data := make([]uint8, size - 8)
	p.Data = data

	for len(data) > 0 {
		n, err := c.conn.Read(data)

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
	binary.LittleEndian.PutUint32(data[4:8], p.Type)
	copy(data[8:], p.Data)

	if c.ocipher != nil {
		c.ocipher.XORKeyStream(data, data)
	}

	for len(data) > 0 {
		w, err := c.conn.Write(data)

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
	return c.conn.Close()
}

func (c *Connection) String() string {
	return fmt.Sprintf("[pso2/net/connection: %s -> %s]", c.conn.LocalAddr(), c.conn.RemoteAddr())
}
