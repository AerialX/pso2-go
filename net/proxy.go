package net

import (
	"net"
	"fmt"
	"sync"
	"crypto/rsa"
	"aaronlindsay.com/go/pkg/pso2/net/packets"
)

type Proxy struct {
	ServerEndpoint, ClientEndpoint string

	connections map[*Connection]*Connection
	connectionsLock sync.Mutex
}

func NewProxy(serverEndpoint, clientEndpoint string) *Proxy {
	return &Proxy{serverEndpoint, clientEndpoint, make(map[*Connection]*Connection), sync.Mutex{}}
}

func (p *Proxy) Listen() (net.Listener, error) {
	return net.Listen("tcp4", p.ServerEndpoint)
}

func (p *Proxy) Start(l net.Listener, serverRoute *PacketRoute, clientRoute *PacketRoute) error {
	for {
		conn, err := l.Accept()

		if err != nil {
			Logger.Errorf("%s %s listener error. %s", p, l, err)
			return err
		}

		go func() {
			Logger.Infof("%s new connection from %s", p, conn.RemoteAddr())
			c := NewConnection(conn)
			client, err := p.Connect(c)
			if err != nil {
				Logger.Errorf("%s %s connection failed. %s", p, c, err)
				c.Close()
			} else {
				p.Proxy(c, serverRoute, client, clientRoute)
				c.Close()
				client.Close()
			}
		}()
	}
}

func (p *Proxy) Connect(c *Connection) (*Connection, error) {
	clientConn, err := net.Dial("tcp4", p.ClientEndpoint)

	if err != nil {
		return nil, err
	}

	return NewConnection(clientConn), nil
}

func (p *Proxy) Proxy(server *Connection, serverRoute *PacketRoute, client *Connection, clientRoute *PacketRoute) error {
	Logger.Infof("%s Proxying connection %s", p, server)

	ch := make(chan error)

	k := func(c *Connection, r *PacketRoute) {
		err := c.RoutePackets(r)
		ch <- err
	}

	p.connectionsLock.Lock()
	p.connections[server] = client
	p.connections[client] = server
	p.connectionsLock.Unlock()

	go k(server, serverRoute)
	go k(client, clientRoute)

	var err error
	for i := 0; i < 2; i++ {
		e := <-ch
		if err == nil {
			err = e
		}

		server.Close()
		client.Close()
	}

	p.connectionsLock.Lock()
	delete(p.connections, server)
	delete(p.connections, client)
	p.connectionsLock.Unlock()

	Logger.Infof("%s Proxy %s closed. %s", p, server, err)

	return err
}

func (p *Proxy) Destination(c *Connection) (d *Connection) {
	p.connectionsLock.Lock()
	d = p.connections[c]
	p.connectionsLock.Unlock()
	return
}

func (p *Proxy) String() string {
	return fmt.Sprintf("[pso2/net/proxy: %s -> %s]", p.ServerEndpoint, p.ClientEndpoint)
}

type ProxyEndpointListener interface {
	EndpointAnnouncement(ip net.IP, port uint16)
}

func ProxyHandlerShip(p *Proxy, l ProxyEndpointListener, ip net.IP) PacketHandler {
	return proxyHandlerShip{p, l, ip}
}

type proxyHandlerShip struct {
	proxy *Proxy
	listener ProxyEndpointListener
	addr net.IP
}

func (h proxyHandlerShip) HandlePacket(c *Connection, p *packets.Packet) (bool, error) {
	Logger.Debugf("%s %s ship packet, rewriting addresses", h.proxy, c)

	s, err := packets.ParseShip(p)
	if err != nil {
		return false, err
	}

	for i := range s.Entries {
		e := &s.Entries[i]
		h.listener.EndpointAnnouncement(e.Address(), 12000 + (uint16(e.Number) % 10000))
		e.SetAddress(h.addr)
	}

	p, err = s.Packet()

	if err != nil {
		return false, err
	}

	return true, h.proxy.Destination(c).WritePacket(p)
}

func ProxyHandlerBlocks(p *Proxy, l ProxyEndpointListener, ip net.IP) PacketHandler {
	return proxyHandlerBlocks{p, l, ip}
}

type proxyHandlerBlocks struct {
	proxy *Proxy
	listener ProxyEndpointListener
	addr net.IP
}

func (h proxyHandlerBlocks) HandlePacket(c *Connection, p *packets.Packet) (bool, error) {
	Logger.Debugf("%s %s block list packet, rewriting addresses", h.proxy, c)

	b, err := packets.ParseBlocks(p)
	if err != nil {
		return false, err
	}

	for i := range b.Entries {
		e := &b.Entries[i]
		h.listener.EndpointAnnouncement(e.Address(), e.Port)
		e.SetAddress(h.addr)
	}

	packetType := p.Type
	p, err = b.Packet()

	if err != nil {
		return false, err
	}

	p.Type = packetType

	return true, h.proxy.Destination(c).WritePacket(p)
}

func ProxyHandlerBlockResponse(p *Proxy, l ProxyEndpointListener, ip net.IP) PacketHandler {
	return proxyHandlerBlockResponse{p, l, ip}
}

type proxyHandlerBlockResponse struct {
	proxy *Proxy
	listener ProxyEndpointListener
	addr net.IP
}

func (h proxyHandlerBlockResponse) HandlePacket(c *Connection, p *packets.Packet) (bool, error) {
	Logger.Debugf("%s %s block response packet, rewriting address", h.proxy, c)

	b, err := packets.ParseBlockResponse(p)
	if err != nil {
		return false, err
	}

	h.listener.EndpointAnnouncement(b.Address(), b.Port)
	b.SetAddress(h.addr)

	p, err = b.Packet()

	if err != nil {
		return false, err
	}

	return true, h.proxy.Destination(c).WritePacket(p)
}

func ProxyHandlerBlock(p *Proxy, l ProxyEndpointListener, ip net.IP) PacketHandler {
	return proxyHandlerBlock{p, l, ip}
}

type proxyHandlerBlock struct {
	proxy *Proxy
	listener ProxyEndpointListener
	addr net.IP
}

func (h proxyHandlerBlock) HandlePacket(c *Connection, p *packets.Packet) (bool, error) {
	Logger.Debugf("%s %s block packet, rewriting address", h.proxy, c)

	b, err := packets.ParseBlock(p)
	if err != nil {
		return false, err
	}

	h.listener.EndpointAnnouncement(b.Address(), b.Port)
	b.SetAddress(h.addr)

	p, err = b.Packet()

	if err != nil {
		return false, err
	}

	return true, h.proxy.Destination(c).WritePacket(p)
}

func ProxyHandlerRoom(p *Proxy, l ProxyEndpointListener, ip net.IP) PacketHandler {
	return proxyHandlerRoom{p, l, ip}
}

type proxyHandlerRoom struct {
	proxy *Proxy
	listener ProxyEndpointListener
	addr net.IP
}

func (h proxyHandlerRoom) HandlePacket(c *Connection, p *packets.Packet) (bool, error) {
	Logger.Debugf("%s %s room packet, rewriting address", h.proxy, c)

	r, err := packets.ParseRoom(p)
	if err != nil {
		return false, err
	}

	h.listener.EndpointAnnouncement(r.Address(), r.Port)
	r.SetAddress(h.addr)

	packetType := p.Type
	p, err = r.Packet()

	if err != nil {
		return false, err
	}

	p.Type = packetType

	return true, h.proxy.Destination(c).WritePacket(p)
}

func ProxyHandlerCipher(p *Proxy, privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) PacketHandler {
	return proxyHandlerCipher{p, privateKey, publicKey}
}

type proxyHandlerCipher struct {
	proxy *Proxy
	privateKey *rsa.PrivateKey
	publicKey *rsa.PublicKey
}

func (h proxyHandlerCipher) HandlePacket(c *Connection, p *packets.Packet) (bool, error) {
	Logger.Debugf("%s %s cipher packet, re-encrypting", h.proxy, c)

	b, err := packets.ParseCipher(p)
	if err != nil {
		return false, err
	}

	key, err := b.Key(h.privateKey)
	if err != nil {
		return false, err
	}

	rc4key, err := packets.CipherRC4Key(key)
	if err != nil {
		return false, err
	}

	b.SetKey(key, h.publicKey)

	p, err = b.Packet()
	if err != nil {
		return false, err
	}

	dest := h.proxy.Destination(c)
	err = dest.WritePacket(p)
	if err == nil {
		err = dest.SetCipher(rc4key)
	}

	return true, err
}

func ProxyHandlerFallback(p *Proxy) PacketHandler {
	return proxyHandlerFallback{p}
}

type proxyHandlerFallback struct {
	proxy *Proxy
}

func (h proxyHandlerFallback) HandlePacket(c *Connection, p *packets.Packet) (bool, error) {
	Logger.Tracef("%s %s unknown packet %s, forwarding", h.proxy, c, p)

	return true, h.proxy.Destination(c).WritePacket(p)
}
