#!/bin/bash
# Capture the 9front VM screen. Output is PNG regardless of extension.
# usage: vm-shot.sh /path/to/out.png
set -e
out="${1:?usage: vm-shot.sh out.png}"
virsh -c qemu:///session screenshot 9front "$out" >/dev/null
echo "$out"
