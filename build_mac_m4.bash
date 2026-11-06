#!/usr/bin/env bash
# Backward-compatibility wrapper for build_mac_m4.bash
echo "ℹ️  build_mac_m4.bash is deprecated, forwarding to ./build.sh..."
exec "$(dirname "$0")/build.sh" "$@"
