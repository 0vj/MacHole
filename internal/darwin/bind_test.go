package darwin

import (
	"testing"
)

func TestDetectPhysicalInterfaces(t *testing.T) {
	ifaces, err := GetPhysicalInterfaces()
	if err != nil {
		t.Fatalf("GetPhysicalInterfaces failed: %v", err)
	}
	t.Logf("Found %d candidate physical interfaces", len(ifaces))
	for _, iface := range ifaces {
		t.Logf(" - %s (Index: %d, IP: %s, Up: %v)", iface.Name, iface.Index, iface.IP, iface.IsUp)
	}
}

func TestDetectDefaultPhysicalInterface(t *testing.T) {
	defIface, err := DetectDefaultPhysicalInterface()
	if err != nil {
		t.Logf("DetectDefaultPhysicalInterface non-fatal error: %v, returned %s", err, defIface)
	}
	if defIface == "" {
		t.Errorf("Expected non-empty default interface")
	}
	t.Logf("Detected default physical interface: %s", defIface)
}
