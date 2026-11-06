---
name: Bug Report
about: Create a report to help us improve MacHole
title: '[BUG] '
labels: bug
assignees: ''
---

**Describe the bug**
A clear and concise description of what the bug is.

**Environment:**
- macOS Version: [e.g. Sonoma 14.5, Sequoia 15.0]
- Chip / Architecture: [e.g. Apple M1, M2, M3, M4 Pro, Intel x86_64]
- Active VPN / Tunnel Client: [e.g. WireGuard, Cisco AnyConnect, Clash Verge, Sing-box TUN]
- Primary Physical Interface: [e.g. en0 (Wi-Fi), en1 (Ethernet)]

**To Reproduce**
Steps to reproduce the behavior:
1. Launch MacHole (`./MacHole` or `MacHole.app`)
2. Configure application SOCKS5 to `127.0.0.1:1080`
3. Test request: `curl --socks5 127.0.0.1:1080 https://api.ipify.org`
4. See error

**Expected behavior**
A clear and concise description of what you expected to happen.

**Logs / Console Output**
If applicable, paste terminal output or lines from the MacHole "Live Socket Event Log":
```text
[paste logs here]
```
