package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"machole/internal/darwin"
)

type dnsCacheEntry struct {
	ip      string
	expires time.Time
}

var (
	dnsCache    sync.Map
	dnsCacheTTL = 5 * time.Minute
)

// ResolveBypassVPN resolves hostnames using an upstream DNS server bound directly to the specified hardware interface.
func ResolveBypassVPN(ctx context.Context, host string, ifaceIndex int, upstreamDNS string) (string, error) {
	if ip := net.ParseIP(host); ip != nil {
		return host, nil
	}

	if val, ok := dnsCache.Load(host); ok {
		entry := val.(dnsCacheEntry)
		if time.Now().Before(entry.expires) {
			return entry.ip, nil
		}
		dnsCache.Delete(host)
	}

	if upstreamDNS == "" {
		upstreamDNS = "1.1.1.1:53"
	}

	dialer := darwin.NewBoundDialer(ifaceIndex)
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, "udp4", upstreamDNS)
		},
	}

	ips, err := resolver.LookupIP(ctx, "ip4", host)
	if err != nil || len(ips) == 0 {
		ips, err = net.DefaultResolver.LookupIP(ctx, "ip4", host)
		if err != nil || len(ips) == 0 {
			return "", fmt.Errorf("DNS lookup failed for %s: %v", host, err)
		}
	}

	resolved := ips[0].String()
	dnsCache.Store(host, dnsCacheEntry{
		ip:      resolved,
		expires: time.Now().Add(dnsCacheTTL),
	})
	return resolved, nil
}
