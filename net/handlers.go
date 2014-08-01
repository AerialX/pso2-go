package net

import (
	"path"
	"fmt"
	"net"
	"os"
	"encoding/binary"
	"crypto/rsa"
	"aaronlindsay.com/go/pkg/pso2/net/packets"
)

func HandlerCipher(privateKey *rsa.PrivateKey) PacketHandler {
	return handlerCipher{privateKey}
}

type handlerCipher struct {
	privateKey *rsa.PrivateKey
}

func (h handlerCipher) HandlePacket(c *Connection, p *packets.Packet) (bool, error) {
	Logger.Debugf("%s cipher packet, enabling encryption", c)

	b, err := packets.ParseCipher(p)
	if err != nil {
		return false, err
	}

	key, err := b.RC4Key(h.privateKey)
	if err != nil {
		return false, err
	}

	err = c.SetCipher(key)

	return true, err
}

func HandlerIgnore(handler PacketHandler) PacketHandler {
	return handlerIgnore{handler}
}

type handlerIgnore struct {
	handler PacketHandler
}

func (h handlerIgnore) HandlePacket(c *Connection, p *packets.Packet) (bool, error) {
	_, err := h.handler.HandlePacket(c, p)
	return false, err
}

func HandlerDump(location string) PacketHandler {
	return handlerDump{location}
}

type handlerDump struct {
	location string
}

func (h handlerDump) HandlePacket(c *Connection, p *packets.Packet) (bool, error) {
	host, portRemote, _ := net.SplitHostPort(c.RemoteAddr().String())
	_, portLocal, _ := net.SplitHostPort(c.LocalAddr().String())
	filename := path.Join(h.location, fmt.Sprintf("%s-%s-%s.dump", host, portLocal, portRemote))

	f, err := os.OpenFile(filename, os.O_WRONLY | os.O_CREATE | os.O_APPEND, 0666)

	if err != nil {
		Logger.Warningf("%s error opening dump file %s. %s", c, filename, err)
		return false, nil
	}

	end := binary.LittleEndian
	binary.Write(f, end, uint32(8 + len(p.Data)))
	binary.Write(f, end, p.Type)
	binary.Write(f, end, p.Flags)
	binary.Write(f, end, p.Data)
	f.Close()

	return false, nil
}
