//go:build !darwin || !cgo

package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// HasNativeGUI reports whether Cocoa WKWebView GUI is compiled in.
func HasNativeGUI() bool {
	return false
}

// RunPlatformGUI runs in headless mode when Cocoa is disabled.
func RunPlatformGUI(uiURL string, onTerminate func()) {
	fmt.Println("ℹ️  Native Cocoa GUI is disabled or not supported in this build. Running in headless mode.")
	if uiURL != "" {
		fmt.Printf("🌐 Web Dashboard is accessible at: %s\n", uiURL)
	}
	fmt.Println("Press Ctrl+C to terminate.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	if onTerminate != nil {
		onTerminate()
	}
}
