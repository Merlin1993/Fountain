package nat

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
	"witCon/log"
)

type Interface interface {
	AddMapping(protocol string, extPort, intPort int, name string, lifetime time.Duration) error

	DeleteMapping(protocol string, extPort, intPort int) error

	ExternalIP() (net.IP, error)

	String() string
}

// Parse parses a NAT interface description.
// The following formats are currently accepted.
// Note that mechanism names are not case-sensitive.
//
//     "" or "none"         return nil
//     "extIp:77.12.33.4"   will assume the local machine is reachable on the given IP
//     "any"                uses the first auto-detected mechanism
//     "uPnp"               uses the Universal Plug and Play protocol
//     "pmp"                uses NAT-PMP with an auto-detected gateway address
//     "pmp:192.168.0.1"    uses NAT-PMP with the given gateway address
func Parse(spec string) (Interface, error) {
	var (
		parts = strings.SplitN(spec, ":", 2)
		mech  = strings.ToLower(parts[0])
		ip    net.IP
	)
	if len(parts) > 1 {
		ip = net.ParseIP(parts[1])
		if ip == nil {
			return nil, errors.New("invalid IP address")
		}
	}
	switch mech {
	case "", "none", "off":
		return nil, nil
	case "any", "auto", "on":
		return Any(), nil
	case "extIp", "ip":
		if ip == nil {
			return nil, errors.New("missing IP address")
		}
		return ExtIP(ip), nil
	case "uPnp":
		return UPnP(), nil
	case "pmp", "natPmp", "nat-pmp":
		return PMP(ip), nil
	default:
		return nil, fmt.Errorf("unknown mechanism %q", parts[0])
	}
}

func Map(m Interface, c chan struct{}, protocol string, extPort, intPort int, name string) {
	refresh := time.NewTimer(mapUpdateInterval)
	defer func() {
		refresh.Stop()
		m.DeleteMapping(protocol, extPort, intPort)
	}()

	if err := m.AddMapping(protocol, extPort, intPort, name, mapTimeout); err != nil {
		log.Debug("Couldn't add port mapping", "err", err)
	} else {
		log.Debug("Mapped network port")
	}

	for {
		select {
		case _, ok := <-c:
			if !ok {
				return
			}
		case <-refresh.C:
			log.Debug("Refreshing port mapping")
			if err := m.AddMapping(protocol, extPort, intPort, name, mapTimeout); err != nil {
				log.Debug("Couldn't add port mapping", "err", err)
			}
			refresh.Reset(mapUpdateInterval)
		}
	}
}

// Any returns a port mapper that tries to discover any supported mechanism on the local network.
func Any() Interface {
	// TODO: attempt to discover whether the local machine has an
	// Internet-class address. Return ExtIP in this case.
	return startAutoDisc("UPnP or NAT-PMP", func() Interface {
		found := make(chan Interface, 2)
		go func() { found <- discoverUPnP() }()
		go func() { found <- discoverPMP() }()
		for i := 0; i < cap(found); i++ {
			if c := <-found; c != nil {
				return c
			}
		}
		return nil
	})
}
