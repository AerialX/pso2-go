package main

import (
	"net"
	"flag"
	"os"
	"fmt"
	"crypto/rsa"
	pso2net "aaronlindsay.com/go/pkg/pso2/net"
	"aaronlindsay.com/go/pkg/pso2/net/packets"
	"github.com/juju/loggo"
)

var Logger loggo.Logger = loggo.GetLogger("pso2.cmd.pso2-net")

func usage() {
	fmt.Fprintln(os.Stderr, "usage: pso2-net [flags]")
	flag.PrintDefaults()
	os.Exit(2)
}

func ragequit(apath string, err error) {
	if err != nil {
		if apath != "" {
			Logger.Errorf("error with file %s", apath)
		}
		Logger.Errorf("%s", err)
		os.Exit(1)
	}
}

func findaddr() (addr string) {
	addr = "127.0.0.1"

	as, err := net.InterfaceAddrs()
	if err != nil {
		return
	}

	for _, a := range as {
		if ip, ok := a.(*net.IPNet); ok {
			ip := ip.IP.To4()
			if ip != nil && !ip.IsLoopback() && !ip.IsMulticast() {
				return ip.String()
			}
		}
	}

	return
}

func main() {
	var flagPrivateKey, flagPublicKey, flagIP, flagLog, flagDump string
	var keyPrivate *rsa.PrivateKey
	var keyPublic *rsa.PublicKey

	flag.Usage = usage
	flag.StringVar(&flagPrivateKey, "priv", "", "server private key")
	flag.StringVar(&flagPublicKey, "pub", "", "client public key")
	flag.StringVar(&flagLog, "log", "info", "log level (trace, debug, info, warning, error, critical)")
	flag.StringVar(&flagIP, "a", findaddr(), "external IPv4 address")
	flag.StringVar(&flagDump, "d", "", "dump packets to folder")
	flag.Parse()

	ip := net.IPv4(127, 0, 0, 1)
	if flagIP != "" {
		ip = net.ParseIP(flagIP)
	}

	if flagLog != "" {
		lvl, ok := loggo.ParseLevel(flagLog)
		if ok {
			Logger.SetLogLevel(lvl)
		} else {
			Logger.Warningf("Invalid log level %s specified", flagLog)
		}
	}
	pso2net.Logger.SetLogLevel(Logger.LogLevel())

	if flagPrivateKey != "" {
		Logger.Infof("Loading private key")
		f, err := os.Open(flagPrivateKey)
		ragequit(flagPrivateKey, err)

		keyPrivate, err = pso2net.LoadPrivateKey(f)
		f.Close()

		ragequit(flagPrivateKey, err)
	}

	if flagPublicKey != "" {
		Logger.Infof("Loading public key")
		f, err := os.Open(flagPublicKey)
		ragequit(flagPublicKey, err)

		keyPublic, err = pso2net.LoadPublicKey(f)
		f.Close()
		ragequit(flagPublicKey, err)
	}

	Logger.Infof("Starting proxy servers on %s", ip)

	fallbackRoute := func(p *pso2net.Proxy) *pso2net.PacketRoute {
		r := &pso2net.PacketRoute{}
		r.RouteMask(0xffffffff, pso2net.RoutePriorityLow, pso2net.ProxyHandlerFallback(p))
		if flagDump != "" {
			r.RouteMask(0xffffffff, pso2net.RoutePriorityHigh, pso2net.HandlerIgnore(pso2net.HandlerDump(flagDump)))
		}
		return r
	}

	newProxy := func(host string, port int) *pso2net.Proxy {
		return pso2net.NewProxy(fmt.Sprintf(":%d", port), fmt.Sprintf("%s:%d", host, port))
	}

	startProxy := func(p *pso2net.Proxy, s *pso2net.PacketRoute, c *pso2net.PacketRoute) {
		l, err := p.Listen()
		ragequit(p.String(), err)

		go p.Start(l, s, c)
	}

	for i := 0; i < packets.ShipCount; i++ {
		blockPort := 12000 + (100 * i)
		shipPort := blockPort + 99

		// Set up ship proxy, rewrites IPs
		proxy := newProxy(packets.ShipHostnames[i], shipPort)
		route := &pso2net.PacketRoute{}
		route.Route(packets.TypeShip, pso2net.RoutePriorityNormal, pso2net.ProxyHandlerShip(proxy, ip))
		route.RouteMask(0xffffffff, pso2net.RoutePriorityLow, pso2net.ProxyHandlerFallback(proxy))
		if flagDump != "" {
			route.RouteMask(0xffffffff, pso2net.RoutePriorityHigh, pso2net.HandlerIgnore(pso2net.HandlerDump(flagDump)))
		}
		startProxy(proxy, fallbackRoute(proxy), route)

		// Set up block proxy, rewrites IPs
		proxy = newProxy(packets.ShipHostnames[i], blockPort)
		route = &pso2net.PacketRoute{}
		route.Route(packets.TypeBlock, pso2net.RoutePriorityNormal, pso2net.ProxyHandlerBlock(proxy, ip))
		route.RouteMask(0xffffffff, pso2net.RoutePriorityLow, pso2net.ProxyHandlerFallback(proxy))
		if flagDump != "" {
			route.RouteMask(0xffffffff, pso2net.RoutePriorityHigh, pso2net.HandlerIgnore(pso2net.HandlerDump(flagDump)))
		}
		startProxy(proxy, fallbackRoute(proxy), route)

		for b := 1; b < 99; b++ {
			proxy = newProxy(packets.ShipHostnames[i], blockPort + b)

			// Set up client route (messages from the PSO2 server)
			route = &pso2net.PacketRoute{}
			route.RouteMask(0xffffffff, pso2net.RoutePriorityLow, pso2net.ProxyHandlerFallback(proxy))
			if flagDump != "" {
				route.RouteMask(0xffffffff, pso2net.RoutePriorityHigh, pso2net.HandlerIgnore(pso2net.HandlerDump(flagDump)))
			}

			// Set up server route (messages from the client)
			sroute := &pso2net.PacketRoute{}
			sroute.Route(packets.TypeCipher, pso2net.RoutePriorityHigh, pso2net.HandlerIgnore(pso2net.HandlerCipher(keyPrivate)))
			sroute.Route(packets.TypeCipher, pso2net.RoutePriorityNormal, pso2net.ProxyHandlerCipher(proxy, keyPrivate, keyPublic))
			sroute.RouteMask(0xffffffff, pso2net.RoutePriorityLow, pso2net.ProxyHandlerFallback(proxy))
			if flagDump != "" {
				sroute.RouteMask(0xffffffff, pso2net.RoutePriorityHigh, pso2net.HandlerIgnore(pso2net.HandlerDump(flagDump)))
			}

			startProxy(proxy, sroute, route)
		}
	}

	// Stop foreverz
	<-make(chan int)
}
