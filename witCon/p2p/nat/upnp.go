package nat

import (
	"errors"
	"fmt"
	"github.com/huin/goupnp"
	"github.com/huin/goupnp/dcps/internetgateway1"
	"github.com/huin/goupnp/dcps/internetgateway2"
	"net"
	"strings"
	"time"
)

type uPnpClient interface {
	GetExternalIPAddress() (string, error)
	AddPortMapping(string, uint16, string, uint16, string, bool, string, uint32) error
	DeletePortMapping(string, uint16, string) error
	GetNATRSIPStatus() (sip bool, nat bool, err error)
}

func UPnP() Interface {
	return startAutoDisc("uPnp", discoverUPnP)
}

type uPnp struct {
	dev     *goupnp.RootDevice
	service string
	client  uPnpClient
}

func (n *uPnp) ExternalIP() (addr net.IP, err error) {
	ipString, err := n.client.GetExternalIPAddress()
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(ipString)
	if ip == nil {
		return nil, errors.New("bad IP in response")
	}
	return ip, nil
}

func (n *uPnp) internalAddress() (net.IP, error) {
	devAddr, err := net.ResolveUDPAddr("udp4", n.dev.URLBase.Host)
	if err != nil {
		return nil, err
	}
	iFaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iFace := range iFaces {
		addrs, err := iFace.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			if x, ok := addr.(*net.IPNet); ok && x.Contains(devAddr.IP) {
				return x.IP, nil
			}
		}
	}
	return nil, fmt.Errorf("could not find local address in same net as %v", devAddr)
}

func (n *uPnp) AddMapping(protocol string, extPort, intPort int, desc string, lifetime time.Duration) error {
	ip, err := n.internalAddress()
	if err != nil {
		return nil
	}
	protocol = strings.ToUpper(protocol)
	lifetimeS := uint32(lifetime / time.Second)
	n.DeleteMapping(protocol, extPort, intPort)
	return n.client.AddPortMapping("", uint16(extPort), protocol, uint16(intPort), ip.String(), true, desc, lifetimeS)
}

func (n *uPnp) DeleteMapping(protocol string, extPort, intPort int) error {
	return n.client.DeletePortMapping("", uint16(extPort), strings.ToUpper(protocol))
}

func (n *uPnp) String() string {
	return "uPnp " + n.service
}

// discoverUPnP searches for Internet Gateway Devices and returns the first one it can find on the local network.
func discoverUPnP() Interface {
	found := make(chan *uPnp, 2)
	// IGDv1
	go discover(found, internetgateway1.URN_WANConnectionDevice_1, func(dev *goupnp.RootDevice, sc goupnp.ServiceClient) *uPnp {
		switch sc.Service.ServiceType {
		case internetgateway1.URN_WANIPConnection_1:
			return &uPnp{dev, "IGDv1-IP1", &internetgateway1.WANIPConnection1{ServiceClient: sc}}
		case internetgateway1.URN_WANPPPConnection_1:
			return &uPnp{dev, "IGDv1-PPP1", &internetgateway1.WANPPPConnection1{ServiceClient: sc}}
		}
		return nil
	})
	// IGDv2
	go discover(found, internetgateway2.URN_WANConnectionDevice_2, func(dev *goupnp.RootDevice, sc goupnp.ServiceClient) *uPnp {
		switch sc.Service.ServiceType {
		case internetgateway2.URN_WANIPConnection_1:
			return &uPnp{dev, "IGDv2-IP1", &internetgateway2.WANIPConnection1{ServiceClient: sc}}
		case internetgateway2.URN_WANIPConnection_2:
			return &uPnp{dev, "IGDv2-IP2", &internetgateway2.WANIPConnection2{ServiceClient: sc}}
		case internetgateway2.URN_WANPPPConnection_1:
			return &uPnp{dev, "IGDv2-PPP1", &internetgateway2.WANPPPConnection1{ServiceClient: sc}}
		}
		return nil
	})
	for i := 0; i < cap(found); i++ {
		if c := <-found; c != nil {
			return c
		}
	}
	return nil
}

// finds devices matching the given target and calls matcher for all
// advertised services of each device. The first non-nil service found
// is sent into out. If no service matched, nil is sent.
func discover(out chan<- *uPnp, target string, matcher func(*goupnp.RootDevice, goupnp.ServiceClient) *uPnp) {
	devs, err := goupnp.DiscoverDevices(target)
	if err != nil {
		out <- nil
		return
	}
	found := false
	for i := 0; i < len(devs) && !found; i++ {
		if devs[i].Root == nil {
			continue
		}
		devs[i].Root.Device.VisitServices(func(service *goupnp.Service) {
			if found {
				return
			}
			// check for a matching IGD service
			sc := goupnp.ServiceClient{
				SOAPClient: service.NewSOAPClient(),
				RootDevice: devs[i].Root,
				Location:   devs[i].Location,
				Service:    service,
			}
			sc.SOAPClient.HTTPClient.Timeout = soapRequestTimeout
			uPnp := matcher(devs[i].Root, sc)
			if uPnp == nil {
				return
			}
			// check whether port mapping is enabled
			if _, nat, err := uPnp.client.GetNATRSIPStatus(); err != nil || !nat {
				return
			}
			out <- uPnp
			found = true
		})
	}
	if !found {
		out <- nil
	}
}
