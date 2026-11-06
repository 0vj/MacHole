package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"machole/internal/app"
	"machole/internal/darwin"
	"machole/internal/proxy"
	"machole/internal/server"
)

func main() {
	headlessFlag := flag.Bool("headless", false, "Run in headless CLI mode without Cocoa GUI")
	listenFlag := flag.String("listen", "127.0.0.1:1080", "SOCKS5 proxy listen address")
	interfaceFlag := flag.String("interface", "", "Hardware interface name (e.g. en0); defaults to active physical default")
	uiAddrFlag := flag.String("ui-addr", "127.0.0.1:19080", "Embedded Web UI / control API address")
	noUIFlag := flag.Bool("no-ui", false, "Disable the embedded Web UI and HTTP API")
	dnsFlag := flag.String("dns", "1.1.1.1:53", "Upstream DNS server to resolve domains via physical interface")
	versionFlag := flag.Bool("version", false, "Print MacHole version and exit")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("MacHole v%s (%s/%s)\n", proxy.Version, runtime.GOOS, runtime.GOARCH)
		return
	}

	p := proxy.NewServer(*dnsFlag)

	selectedIface := *interfaceFlag
	if selectedIface == "" {
		detected, err := darwin.DetectDefaultPhysicalInterface()
		if err == nil && detected != "" {
			selectedIface = detected
		} else {
			selectedIface = "en0"
		}
	}

	if err := p.Start(*listenFlag, selectedIface); err != nil {
		log.Printf("⚠️  Warning: Initial proxy startup on %s (%s) failed: %v", *listenFlag, selectedIface, err)
	}

	var uiURL string
	if !*noUIFlag {
		uiURL = "http://" + *uiAddrFlag
		httpSrv := server.NewHTTPServer(*uiAddrFlag, p)
		httpSrv.Start()
		p.AddLog("GUI Dashboard initialized on %s", uiURL)
	}

	if *headlessFlag || !app.HasNativeGUI() {
		fmt.Printf("🚀 MacHole v%s is running in headless mode.\n", proxy.Version)
		fmt.Printf("📡 SOCKS5 Proxy : %s (Bound to %s)\n", *listenFlag, selectedIface)
		if !*noUIFlag {
			fmt.Printf("🌐 Web Dashboard: %s\n", uiURL)
		}
		fmt.Println("Press Ctrl+C to terminate.")

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		fmt.Println("\nShutting down MacHole gracefully...")
		p.Stop()
		return
	}

	app.RunPlatformGUI(uiURL, func() {
		p.Stop()
	})
}
