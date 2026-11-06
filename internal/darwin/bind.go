package darwin

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// macOS Darwin BSD socket constants for binding a socket directly to a hardware interface index.
// Defined in <netinet/in.h>: #define IP_BOUND_IF 25
// And for IPv6 in <netinet6/in6.h>: #define IPV6_BOUND_IF 125
const (
	IP_BOUND_IF   = 25
	IPV6_BOUND_IF = 125
)

// InterfaceInfo represents a network interface's state.
type InterfaceInfo struct {
	Name  string `json:"name"`
	Index int    `json:"index"`
	IP    string `json:"ip"`
	IsUp  bool   `json:"is_up"`
}

// NewBoundDialer returns a net.Dialer whose sockets are pinned to a specific hardware interface index via Darwin setsockopt.
func NewBoundDialer(ifaceIndex int) *net.Dialer {
	return &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(network, address string, c syscall.RawConn) error {
			var sockErr error
			err := c.Control(func(fd uintptr) {
				if strings.HasSuffix(network, "6") {
					sockErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, IPV6_BOUND_IF, ifaceIndex)
				} else {
					sockErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, IP_BOUND_IF, ifaceIndex)
				}
			})
			if err != nil {
				return err
			}
			return sockErr
		},
	}
}

// GetPhysicalInterfaces returns non-virtual, non-tunnel network interfaces.
func GetPhysicalInterfaces() ([]InterfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var result []InterfaceInfo
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if strings.HasPrefix(iface.Name, "utun") ||
			strings.HasPrefix(iface.Name, "ipsec") ||
			strings.HasPrefix(iface.Name, "ppp") ||
			strings.HasPrefix(iface.Name, "gif") ||
			strings.HasPrefix(iface.Name, "stf") ||
			strings.HasPrefix(iface.Name, "awdl") ||
			strings.HasPrefix(iface.Name, "llw") ||
			strings.HasPrefix(iface.Name, "anpi") ||
			strings.HasPrefix(iface.Name, "ap") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		var ipv4 string
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				ipv4 = ipnet.IP.String()
				break
			}
		}
		if ipv4 != "" || strings.HasPrefix(iface.Name, "en") {
			result = append(result, InterfaceInfo{
				Name:  iface.Name,
				Index: iface.Index,
				IP:    ipv4,
				IsUp:  (iface.Flags&net.FlagUp != 0) && ipv4 != "",
			})
		}
	}
	return result, nil
}

// DetectDefaultPhysicalInterface returns the default physical route interface, ignoring VPN tunnels.
func DetectDefaultPhysicalInterface() (string, error) {
	out, err := exec.Command("route", "-n", "get", "default").Output()
	if err == nil {
		re := regexp.MustCompile(`interface:\s+(\S+)`)
		matches := re.FindSubmatch(out)
		if len(matches) > 1 {
			ifaceName := string(matches[1])
			if !strings.HasPrefix(ifaceName, "utun") && !strings.HasPrefix(ifaceName, "ppp") && !strings.HasPrefix(ifaceName, "ipsec") {
				return ifaceName, nil
			}
		}
	}

	ifaces, err := GetPhysicalInterfaces()
	if err != nil {
		return "en0", err
	}
	for _, iface := range ifaces {
		if iface.IsUp && iface.IP != "" {
			return iface.Name, nil
		}
	}
	return "en0", fmt.Errorf("no active physical interface found with IPv4")
}
