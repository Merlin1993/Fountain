package netutil

import (
	"fmt"
	"net"
	"testing"
)

func TestDistinctNetSet(t *testing.T) {
	ops := []struct {
		add, remove string
		fails       bool
	}{
		{add: "127.0.0.1"},
		{add: "127.0.0.2"},
		{add: "127.0.0.3", fails: true},
		{add: "127.32.0.1"},
		{add: "127.32.0.2"},
		{add: "127.32.0.3", fails: true},
		{add: "127.33.0.1", fails: true},
		{add: "127.34.0.1"},
		{add: "127.34.0.2"},
		{add: "127.34.0.3", fails: true},
		// Make room for an address, then add again.
		{remove: "127.0.0.1"},
		{add: "127.0.0.3"},
		{add: "127.0.0.3", fails: true},
	}
	set := DistinctNetSet{Subnet: 15, Limit: 2}
	for _, op := range ops {
		var desc string
		if op.add != "" {
			desc = fmt.Sprintf("Add(%s)", op.add)
			if ok := set.Add(parseIP(op.add)); ok != !op.fails {
				t.Errorf("%s == %t, want %t", desc, ok, !op.fails)
			}
		} else {
			desc = fmt.Sprintf("Remove(%s)", op.remove)
			set.Remove(parseIP(op.remove))
		}
		t.Logf("%s: %v", desc, set)
	}
}

func parseIP(s string) net.IP {
	ip := net.ParseIP(s)
	if ip == nil {
		panic("invalid " + s)
	}
	return ip
}

func TestLocalIPv4s(t *testing.T) {
	ip, err := LocalIPv4s()
	if err != nil {
		t.Log("get ipv4 err", "err", err)
	} else {
		t.Log("get local ipv4 : ", "ip", ip)
	}
}
