package nat

import (
	"fmt"
	natpmp "github.com/jackpal/go-nat-pmp"
	"net"
	"strings"
	"time"
)

func PMP(gateway net.IP) Interface {
	if gateway != nil {
		return &pmp{gw: gateway, client: natpmp.NewClient(gateway)}
	}
	return startAutoDisc("NAT-PMP", discoverPMP)
}

type pmp struct {
	gw net.IP
	client  *natpmp.Client
}

func (pmp *pmp) String() string {
	return fmt.Sprintf("NAT-PMP(%v)", pmp.gw)
}

func (pmp *pmp) ExternalIP() (net.IP, error) {
	response, err := pmp.client.GetExternalAddress()
	if err != nil {
		return nil, err
	}
	return response.ExternalIPAddress[:], nil
}

func (pmp *pmp) AddMapping(protocol string, extPort, intPort int, name string, lifetime time.Duration) error {
	if lifetime <= 0 {
		return fmt.Errorf("lifetime must not be <= 0")
	}
	_, err := pmp.client.AddPortMapping(strings.ToLower(protocol), intPort, extPort, int(lifetime/time.Second))
	return err
}

func (pmp *pmp) DeleteMapping(protocol string, extPort, intPort int) (err error) {
	_, err = pmp.client.AddPortMapping(strings.ToLower(protocol), intPort, 0, 0)
	return err
}

func discoverPMP() Interface {
	// run external address lookups on all potential gateways
	gws := potentialGateways()
	found := make(chan *pmp, len(gws))
	for i := range gws {
		gw := gws[i]
		go func() {
			c := natpmp.NewClient(gw)
			if _, err := c.GetExternalAddress(); err != nil {
				found <- nil
			} else {
				found <- &pmp{gw, c}
			}
		}()
	}
	// return the one that responds first.
	// discovery needs to be quick, so we stop caring about
	// any responses after a very short timeout.
	timeout := time.NewTimer(1 * time.Second)
	defer timeout.Stop()
	for range gws {
		select {
		case c := <-found:
			if c != nil {
				return c
			}
		case <-timeout.C:
			return nil
		}
	}
	return nil
}

var (
	// LAN IP ranges
	_, lan10, _  = net.ParseCIDR("10.0.0.0/8")
	_, lan176, _ = net.ParseCIDR("172.16.0.0/12")
	_, lan192, _ = net.ParseCIDR("192.168.0.0/16")
)

func potentialGateways() (gws []net.IP) {
	iFaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, iFace := range iFaces {
		ifAddrs, err := iFace.Addrs()
		if err != nil {
			return gws
		}
		for _, addr := range ifAddrs {
			if x, ok := addr.(*net.IPNet); ok {
				if lan10.Contains(x.IP) || lan176.Contains(x.IP) || lan192.Contains(x.IP) {
					ip := x.IP.Mask(x.Mask).To4()
					if ip != nil {
						ip[3] = ip[3] | 0x01
						gws = append(gws, ip)
					}
				}
			}
		}
	}
	return gws
}
