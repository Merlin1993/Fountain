package nat

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// autoDisc represents a port mapping mechanism that is still being auto-discovered.
// Calls to the Interface methods on this types will
// wait until the discovery is done and then call the method on the discovered mechanism.
type autoDisc struct {
	what string // types of interface being auto-discovered
	once sync.Once
	doIt func() Interface

	mu    sync.Mutex
	found Interface
}

func startAutoDisc(what string, doIt func() Interface) Interface {
	// TODO: monitor network configuration and rerun doit when it changes.
	return &autoDisc{what: what, doIt: doIt}
}

// wait blocks until auto-discovery has been performed.
func (auto autoDisc) wait() error {
	auto.once.Do(func() {
		auto.mu.Lock()
		auto.found = auto.doIt()
		auto.mu.Unlock()
	})
	if auto.found == nil {
		return fmt.Errorf("no %s router discovered", auto.what)
	}
	return nil
}

func (auto autoDisc) AddMapping(protocol string, extPort, intPort int, name string, lifetime time.Duration) error {
	if err := auto.wait(); err != nil {
		return err
	}
	return auto.found.AddMapping(protocol, extPort, intPort, name, lifetime)
}

func (auto autoDisc) DeleteMapping(protocol string, extPort, intPort int) error {
	if err := auto.wait(); err != nil {
		return err
	}
	return auto.found.DeleteMapping(protocol, extPort, intPort)
}

func (auto autoDisc) ExternalIP() (net.IP, error) {
	if err := auto.wait(); err != nil {
		return nil, err
	}
	return auto.found.ExternalIP()
}

func (auto autoDisc) String() string {
	auto.mu.Lock()
	defer auto.mu.Unlock()

	if auto.found == nil {
		return auto.what
	} else {
		return auto.found.String()
	}
}

