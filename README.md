# MacHole

[![Platform](https://img.shields.io/badge/Platform-macOS%2013%2B-000000?logo=apple&logoColor=white)](https://apple.com)
[![Architecture](https://img.shields.io/badge/Architecture-Universal%20(arm64%20%2B%20x86__64)-blue)](#)
[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

**Hardware-Bound SOCKS5 Proxy for macOS (`darwin/arm64` & `darwin/amd64`)**  
Route specific applications directly through your physical network interface (`en0` Wi-Fi / Ethernet) while a system-wide VPN (`utun`) remains active—without root privileges, routing table hacks, or kernel extensions.

---

## Why MacHole?

When you connect to a VPN on macOS (WireGuard, OpenVPN, Cisco AnyConnect, Clash/Sing-box TUN mode, or corporate clients), the VPN captures the default route (`0.0.0.0/0` via `utun*`) and often intercepts system DNS.

If you need certain local or latency-sensitive apps (such as banking portals, local registries, download managers, or a secondary browser) to bypass the VPN and use your direct ISP connection, modifying the macOS routing table manually is brittle and breaks whenever the VPN reconnects.

**MacHole** solves this cleanly at the Darwin socket layer:
- **Kernel-Level Interface Binding (`IP_BOUND_IF`)**: Outbound TCP/UDP sockets created by MacHole set the XNU/Darwin `IP_BOUND_IF` (`25` for IPv4) and `IPV6_BOUND_IF` (`125` for IPv6) socket options via `setsockopt`, pinning packets directly to the hardware interface index (e.g., `en0`) before they ever touch the VPN tunnel.
- **VPN-Bypassed DNS Resolution**: Domain lookups inside SOCKS5 `CONNECT` requests are resolved over a dedicated UDP socket bound to the physical interface (`1.1.1.1:53` with configurable fallback and built-in TTL caching), preventing VPN Fake-IP or DNS hijacking.
- **Dual Mode (Native Cocoa GUI or Headless CLI)**: Launch as a standalone macOS utility with an embedded Dark Aqua WKWebView inspector, or run headlessly as a CLI daemon on servers / SSH sessions.
- **Zero Root Privileges (`No sudo`)**: Binding user-space sockets to a hardware interface index on macOS does not require administrator privileges.
- **Universal Binary**: Builds natively for both Apple Silicon (`arm64` M1/M2/M3/M4) and Intel (`x86_64`) Macs.

---

## Architecture

```text
+--------------------+       SOCKS5 (127.0.0.1:1080)       +-------------------------+
|  Client App        | ----------------------------------> |  MacHole SOCKS5 Server  |
|  (Browser/CLI/App) |                                     +------------+------------+
+--------------------+                                                  |
                                                                        | setsockopt(fd, IPPROTO_IP, IP_BOUND_IF, ifindex)
                                                                        v
+-----------------------------------------------------------------------+------------+
| macOS Darwin Kernel (XNU Networking Stack)                            |            |
|                                                                       |            |
|   [ utun0 / utun4 (Active System VPN) ]   <-- (Ignored by MacHole)    |            |
|                                                                       v            |
|   [ en0 (Physical Wi-Fi / Ethernet) ]     <---------------------------+            |
+------------------------------------------------------------------------------------+
                                                                        |
                                                                        v
                                                           Direct ISP Gateway / Internet
```

---

## Quick Start

### Prerequisites
- macOS 13.0+ (Ventura, Sonoma, Sequoia)
- Go 1.22+ (`brew install go`)

### Build Universal App Bundle

```bash
git clone https://github.com/<your-username>/MacHole.git
cd MacHole

# Build Universal binary and MacHole.app
make app
# or: ./build.sh
```

This creates:
- `./MacHole` — Universal binary (`arm64` + `x86_64`).
- `./MacHole.app` — Ad-hoc signed macOS application bundle ready to move to `/Applications`.

---

## Usage

### 1. Native GUI Mode
Double-click **`MacHole.app`** (or run `./MacHole`).
- A macOS utility window opens automatically with the interface inspector and real-time socket telemetry.
- SOCKS5 proxy starts on `127.0.0.1:1080` bound to your active physical interface (`en0` by default).

### 2. Headless / CLI Mode
To run without GUI (e.g. in Terminal, tmux, or launchd):

```bash
./MacHole -headless
```

#### Command-Line Flags
| Flag | Default | Description |
|---|---|---|
| `-headless` | `false` | Run in headless CLI mode without Cocoa GUI |
| `-listen` | `127.0.0.1:1080` | SOCKS5 proxy listen address |
| `-interface` | `auto` | Hardware interface name (e.g. `en0`, `en1`); auto-detects active physical route by default |
| `-ui-addr` | `127.0.0.1:19080` | Web dashboard & HTTP API address |
| `-no-ui` | `false` | Disable the embedded Web UI and HTTP API completely |
| `-dns` | `1.1.1.1:53` | Upstream DNS server to query over physical interface |
| `-version` | `false` | Print version and exit |

---

## Verifying VPN Bypass

With your VPN active, compare your system IP against the IP routed through MacHole:

```bash
# 1. System default (routed through VPN tunnel)
curl https://api.ipify.org

# 2. Routed through MacHole (bypasses VPN directly to ISP)
curl --socks5 127.0.0.1:1080 https://api.ipify.org
```

You can also click **Verify Direct Egress IP** inside the MacHole UI.

---

## Configuring Applications

Configure any application to use the SOCKS5 proxy:
- **Type**: SOCKS5
- **Host**: `127.0.0.1`
- **Port**: `1080`

> **Tip for Web Browsers (Firefox / Chrome / Safari)**:  
> When configuring browser proxy settings, ensure **"Proxy DNS when using SOCKS v5"** is checked. This ensures domain lookups happen inside MacHole and bypass VPN Fake-IP/DNS hijacking.

---

## Project Structure

```text
MacHole/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml              # CI workflow (go vet, unit tests, universal build)
│   │   └── release.yml         # Automated GitHub Releases on version tags
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md       # GitHub Issue bug template
│   │   └── feature_request.md  # GitHub Issue feature template
│   └── pull_request_template.md# GitHub PR template
├── cmd/
│   └── machole/
│       └── main.go             # Application CLI entrypoint & flag parsing
├── internal/
│   ├── app/
│   │   ├── gui_darwin.go       # Cocoa / WKWebView Objective-C wrapper (cgo)
│   │   └── gui_fallback.go     # Headless fallback (pure Go / non-cgo)
│   ├── darwin/
│   │   ├── bind.go             # Darwin socket IP_BOUND_IF & physical interface discovery
│   │   └── bind_test.go        # Interface discovery unit tests
│   ├── proxy/
│   │   ├── dns.go              # Interface-pinned DNS resolver & in-memory cache
│   │   ├── socks5.go           # RFC 1928 SOCKS5 listener & relay engine
│   │   └── proxy_test.go       # Proxy & DNS unit tests
│   └── server/
│       ├── server.go           # Embedded dashboard & REST API controller
│       ├── server_test.go      # Dashboard API unit tests
│       └── ui.html             # Embedded native-style macOS Inspector dashboard
├── build.sh                    # Universal binary + .app bundle packaging script
├── build_mac_m4.bash           # Deprecation wrapper forwarding to build.sh
├── go.mod                      # Go module definition
├── Makefile                    # Build, run, test, and lint targets
├── CONTRIBUTING.md             # Developer contribution guidelines
├── LICENSE                     # MIT License
└── README.md                   # Documentation
```

---

## Development & Testing

```bash
# Run static analysis
make lint

# Run unit tests
make test

# Build native binary only
make build
```

---

## License

[MIT License](LICENSE) © 2026 Amirmahdi
