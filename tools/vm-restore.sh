#!/bin/bash
# Recreate the working disk from a read-only golden image.
# DESTROYS all current VM state.
#   ./tools/vm-restore.sh      # 9front-golden2.qcow2: with host plumbing
#   ./tools/vm-restore.sh 1    # 9front-golden.qcow2:  before the plumbing
set -e
cd "$(dirname "$0")/.."
case "${1:-2}" in
2)	img=9front-golden2.qcow2 ;;
1)	img=9front-golden.qcow2 ;;
*)	echo "usage: vm-restore.sh [1|2]" >&2; exit 2 ;;
esac
./tools/vm-umount.sh >/dev/null 2>&1 || true
virsh -c qemu:///session destroy 9front 2>/dev/null || true
rm -f 9front.qcow2
qemu-img convert -O qcow2 "$img" 9front.qcow2
echo "restored 9front.qcow2 from $img"
