package proxy

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"machole/internal/darwin"
)

// Version is the current release of MacHole.
const Version = "1.1.0"

// Status represents the runtime telemetry of the proxy.
type Status struct {
	Version        string                 `json:"version"`
	Running        bool                   `json:"running"`
	ListenAddr     string                 `json:"listen_addr"`
	BoundInterface string                 `json:"bound_interface"`
	BoundIndex     int                    `json:"bound_index"`
	BoundIP        string                 `json:"bound_ip"`
	UpstreamDNS    string                 `json:"upstream_dns"`
	ActiveConns    int64                  `json:"active_conns"`
	TotalConns     uint64                 `json:"total_conns"`
	BytesTX        uint64                 `json:"bytes_tx"`
	BytesRX        uint64                 `json:"bytes_rx"`
	LastError      string                 `json:"last_error"`
	Interfaces     []darwin.InterfaceInfo `json:"interfaces"`
	RecentLogs     []string               `json:"recent_logs"`
}

// Server is the hardware-bound SOCKS5 proxy engine.
type Server struct {
	mu             sync.Mutex
	listener       net.Listener
	running        bool
	listenAddr     string
	boundInterface string
	boundIndex     int
	boundIP        string
	upstreamDNS    string
	activeConns    atomic.Int64
	totalConns     atomic.Uint64
	bytesTX        atomic.Uint64
	bytesRX        atomic.Uint64
	lastError      string
	logs           []string
}

// NewServer creates a new SOCKS5 proxy server.
func NewServer(upstreamDNS string) *Server {
	if upstreamDNS == "" {
		upstreamDNS = "1.1.1.1:53"
	}
	return &Server{
		listenAddr:     "127.0.0.1:1080",
		boundInterface: "en0",
		upstreamDNS:    upstreamDNS,
		logs:           make([]string, 0, 100),
	}
}

// AddLog appends a timestamped log to the in-memory buffer and standard logger.
func (p *Server) AddLog(format string, args ...any) {
	msg := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
	log.Println(msg)
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.logs) >= 80 {
		p.logs = p.logs[1:]
	}
	p.logs = append(p.logs, msg)
}

// Start launches the SOCKS5 proxy listener bound to the specified interface.
func (p *Server) Start(listenAddr, ifaceName string) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		p.Stop()
		p.mu.Lock()
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		p.lastError = fmt.Sprintf("Interface %s not found: %v", ifaceName, err)
		p.mu.Unlock()
		return err
	}

	var ipv4 string
	addrs, _ := iface.Addrs()
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			ipv4 = ipnet.IP.String()
			break
		}
	}
	if ipv4 == "" {
		err := fmt.Errorf("interface %s has no active IPv4 address", ifaceName)
		p.lastError = err.Error()
		p.mu.Unlock()
		return err
	}

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		p.lastError = fmt.Sprintf("Failed to listen on %s: %v", listenAddr, err)
		p.mu.Unlock()
		return err
	}

	p.listener = ln
	p.running = true
	p.listenAddr = listenAddr
	p.boundInterface = iface.Name
	p.boundIndex = iface.Index
	p.boundIP = ipv4
	p.lastError = ""
	p.mu.Unlock()

	p.AddLog("SOCKS5 proxy listening on %s -> bound to %s [Index: %d, IP: %s] (VPN bypassed)",
		listenAddr, iface.Name, iface.Index, ipv4)

	go p.acceptLoop(ln, iface.Index)
	return nil
}

// Stop shuts down the SOCKS5 proxy listener.
func (p *Server) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	if p.listener != nil {
		_ = p.listener.Close()
		p.listener = nil
	}
	p.mu.Unlock()
	p.AddLog("SOCKS5 proxy stopped.")
}

func (p *Server) acceptLoop(ln net.Listener, ifaceIndex int) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			p.AddLog("Accept error: %v", err)
			continue
		}
		p.totalConns.Add(1)
		p.activeConns.Add(1)
		go func(c net.Conn) {
			defer func() {
				_ = c.Close()
				p.activeConns.Add(-1)
			}()
			p.handleSOCKS5(c, ifaceIndex)
		}(conn)
	}
}

func (p *Server) handleSOCKS5(client net.Conn, ifaceIndex int) {
	_ = client.SetDeadline(time.Now().Add(20 * time.Second))

	header := make([]byte, 2)
	if _, err := io.ReadFull(client, header); err != nil {
		return
	}
	if header[0] != 0x05 {
		return
	}
	numMethods := int(header[1])
	methods := make([]byte, numMethods)
	if _, err := io.ReadFull(client, methods); err != nil {
		return
	}
	if _, err := client.Write([]byte{0x05, 0x00}); err != nil {
		return
	}

	req := make([]byte, 4)
	if _, err := io.ReadFull(client, req); err != nil {
		return
	}
	if req[0] != 0x05 || req[1] != 0x01 {
		_, _ = client.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	var host string
	switch req[3] {
	case 0x01:
		ipBuf := make([]byte, 4)
		if _, err := io.ReadFull(client, ipBuf); err != nil {
			return
		}
		host = net.IP(ipBuf).String()
	case 0x03:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(client, lenBuf); err != nil {
			return
		}
		domainBuf := make([]byte, int(lenBuf[0]))
		if _, err := io.ReadFull(client, domainBuf); err != nil {
			return
		}
		host = string(domainBuf)
	case 0x04:
		ipBuf := make([]byte, 16)
		if _, err := io.ReadFull(client, ipBuf); err != nil {
			return
		}
		host = net.IP(ipBuf).String()
	default:
		_, _ = client.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(client, portBuf); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(portBuf)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resolvedIP, err := ResolveBypassVPN(ctx, host, ifaceIndex, p.upstreamDNS)
	cancel()
	if err != nil {
		p.AddLog("DNS error (%s): %v", host, err)
		_, _ = client.Write([]byte{0x05, 0x04, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	targetAddr := net.JoinHostPort(resolvedIP, strconv.Itoa(int(port)))
	dialer := darwin.NewBoundDialer(ifaceIndex)
	remote, err := dialer.Dial("tcp", targetAddr)
	if err != nil {
		p.AddLog("Connect failed %s (%s): %v", host, targetAddr, err)
		_, _ = client.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer remote.Close()

	_ = client.SetDeadline(time.Time{})

	if _, err := client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}

	p.AddLog("CONNECT %s:%d -> %s (Direct via %s)", host, port, resolvedIP, p.boundInterface)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		n, _ := io.Copy(remote, client)
		if n > 0 {
			p.bytesTX.Add(uint64(n))
		}
		if tcpConn, ok := remote.(*net.TCPConn); ok {
			_ = tcpConn.CloseWrite()
		}
		_ = remote.SetReadDeadline(time.Now().Add(5 * time.Second))
	}()

	go func() {
		defer wg.Done()
		n, _ := io.Copy(client, remote)
		if n > 0 {
			p.bytesRX.Add(uint64(n))
		}
		if tcpConn, ok := client.(*net.TCPConn); ok {
			_ = tcpConn.CloseWrite()
		}
		_ = client.SetReadDeadline(time.Now().Add(5 * time.Second))
	}()

	wg.Wait()
}

// Status returns the current server metrics and interfaces.
func (p *Server) Status() Status {
	ifaces, _ := darwin.GetPhysicalInterfaces()
	p.mu.Lock()
	defer p.mu.Unlock()
	logsCopy := make([]string, len(p.logs))
	copy(logsCopy, p.logs)
	return Status{
		Version:        Version,
		Running:        p.running,
		ListenAddr:     p.listenAddr,
		BoundInterface: p.boundInterface,
		BoundIndex:     p.boundIndex,
		BoundIP:        p.boundIP,
		UpstreamDNS:    p.upstreamDNS,
		ActiveConns:    p.activeConns.Load(),
		TotalConns:     p.totalConns.Load(),
		BytesTX:        p.bytesTX.Load(),
		BytesRX:        p.bytesRX.Load(),
		LastError:      p.lastError,
		Interfaces:     ifaces,
		RecentLogs:     logsCopy,
	}
}

// CheckExternalIP queries api.ipify.org directly over the pinned physical interface.
func (p *Server) CheckExternalIP() (string, error) {
	p.mu.Lock()
	idx := p.boundIndex
	dns := p.upstreamDNS
	p.mu.Unlock()
	if idx == 0 {
		iface, err := net.InterfaceByName(p.boundInterface)
		if err != nil {
			return "", err
		}
		idx = iface.Index
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	ipAddr, err := ResolveBypassVPN(ctx, "api.ipify.org", idx, dns)
	if err != nil {
		return "", err
	}

	dialer := darwin.NewBoundDialer(idx)
	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp", net.JoinHostPort(ipAddr, "443"))
		},
	}
	client := &http.Client{Transport: tr, Timeout: 8 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.ipify.org?format=json", nil)
	req.Host = "api.ipify.org"
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
