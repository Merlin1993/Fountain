package nat

import (
	"fmt"
	"net"
	"time"
)

type ExtIP net.IP

func (n ExtIP) ExternalIP() (net.IP, error)                            { return net.IP(n), nil }
func (n ExtIP) String() string                                         { return fmt.Sprintf("ExtIP(%v)", net.IP(n)) }
func (ExtIP) AddMapping(string, int, int, string, time.Duration) error { return nil }
func (ExtIP) DeleteMapping(string, int, int) error                     { return nil }


