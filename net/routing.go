package net

import (
	"sort"
	"aaronlindsay.com/go/pkg/pso2/net/packets"
)

const (
	RoutePriorityLow int = -100
	RoutePriorityNormal = 0
	RoutePriorityHigh = 100
)

type PacketHandler interface {
	HandlePacket(c *Connection, p *packets.Packet) (bool, error)
}

type packetHandlerItem struct {
	handler PacketHandler
	priority int
}

type packetHandlerMaskItem struct {
	packetHandlerItem
	mask uint32
}

type PacketRoute struct {
	items map[uint32][]packetHandlerItem
	masks []packetHandlerMaskItem
}

func (r *PacketRoute) Route(t uint32, prio int, h PacketHandler) {
	if r.items == nil {
		r.items = make(map[uint32][]packetHandlerItem)
	}

	r.items[t] = append(r.items[t], packetHandlerItem{h, prio})
}

func (r *PacketRoute) RouteMask(t uint32, prio int, h PacketHandler) {
	r.masks = append(r.masks, packetHandlerMaskItem{packetHandlerItem{h, prio}, t})
}

type packetHandlerList []*packetHandlerItem
func (p packetHandlerList) Len() int {
	return len(p)
}

func (p packetHandlerList) Less(i, j int) bool {
	return p[i].priority > p[j].priority
}

func (p packetHandlerList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (r *PacketRoute) RoutePacket(c *Connection, p *packets.Packet) (consumed bool, err error) {
	items := make(packetHandlerList, 0, 4)

	for i, h := range r.masks {
		if h.mask & p.Type != 0 {
			items = append(items, &r.masks[i].packetHandlerItem)
		}
	}

	if l, ok := r.items[p.Type]; ok {
		for i := range l {
			items = append(items, &l[i])
		}
	}

	sort.Sort(items)

	for _, i := range items {
		consumed, err = i.handler.HandlePacket(c, p)

		if consumed || err != nil {
			return
		}
	}

	return
}
