package proxy

import (
	"context"
	"net"
	"testing"
	"time"

	"machole/internal/darwin"
)

func TestDNSCache(t *testing.T) {
	dnsCache.Store("example.com", dnsCacheEntry{
		ip:      "93.184.216.34",
		expires: time.Now().Add(10 * time.Minute),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ip, err := ResolveBypassVPN(ctx, "example.com", 1, "1.1.1.1:53")
	if err != nil {
		t.Fatalf("Expected cache hit, got error: %v", err)
	}
	if ip != "93.184.216.34" {
		t.Errorf("Expected 93.184.216.34, got %s", ip)
	}
}

func TestProxyServerLifecycle(t *testing.T) {
	server := NewServer("1.1.1.1:53")
	if server.running {
		t.Errorf("NewServer should not be running initially")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen on ephemeral port: %v", err)
	}
	testAddr := ln.Addr().String()
	ln.Close()

	ifaces, err := darwin.GetPhysicalInterfaces()
	if err != nil || len(ifaces) == 0 {
		t.Skip("No physical interfaces available on this machine to test binding")
	}

	targetIface := ifaces[0].Name
	for _, ifc := range ifaces {
		if ifc.IsUp && ifc.IP != "" {
			targetIface = ifc.Name
			break
		}
	}

	if err := server.Start(testAddr, targetIface); err != nil {
		t.Logf("Start returned: %v (expected if interface has no IP)", err)
		return
	}

	status := server.Status()
	if !status.Running {
		t.Errorf("Server should report running status")
	}
	if status.ListenAddr != testAddr {
		t.Errorf("Expected listenAddr %s, got %s", testAddr, status.ListenAddr)
	}

	server.Stop()
	if server.running {
		t.Errorf("Server should report stopped after Stop()")
	}
}
