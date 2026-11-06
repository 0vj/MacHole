# Contributing to MacHole

Thank you for your interest in improving MacHole! We welcome issues, bug reports, documentation enhancements, and pull requests.

## Development Setup

### Prerequisites
- macOS 13.0+ (Ventura, Sonoma, Sequoia)
- Apple Silicon or Intel Mac
- Go 1.22+ (`brew install go`)
- Xcode Command Line Tools (`xcode-select --install`)

### Project Layout

```text
MacHole/
├── cmd/
│   └── machole/
│       └── main.go          # Application CLI & GUI entrypoint
├── internal/
│   ├── app/
│   │   ├── gui_darwin.go    # Objective-C Cocoa / WKWebView wrapper (cgo)
│   │   └── gui_fallback.go  # Pure Go headless fallback
│   ├── darwin/
│   │   ├── bind.go          # Darwin socket IP_BOUND_IF & interface detection
│   │   └── bind_test.go     # Interface discovery tests
│   ├── proxy/
│   │   ├── dns.go           # Interface-pinned DNS resolver & in-memory cache
│   │   ├── socks5.go        # RFC 1928 SOCKS5 listener & relay engine
│   │   └── proxy_test.go    # Proxy & DNS unit tests
│   └── server/
│       ├── server.go        # Embedded dashboard & REST API controller
│       ├── server_test.go   # API tests
│       └── ui.html          # Embedded UI
├── build.sh                 # Universal binary & .app packager
├── Makefile                 # Common developer workflow targets
└── README.md                # Documentation
```

## Workflow Commands

```bash
# Run static analysis
make lint

# Run all unit tests
make test

# Build standard native binary
make build

# Build universal macOS .app bundle
make app

# Run in headless CLI mode
make run-headless

# Clean artifacts
make clean
```

## Pull Request Guidelines

1. Fork the repo and create your feature branch: `git checkout -b feat/my-feature`.
2. Ensure `make lint` and `make test` pass without warnings.
3. Keep commits atomic and write clear commit messages following Conventional Commits (e.g. `feat: ...`, `fix: ...`, `docs: ...`).
4. Submit a Pull Request targeting `main`.
