#!/bin/bash
# Shut down 9front CLEANLY.  hjfs must flush -- never `virsh destroy` a
# running guest or you risk filesystem damage.
set -e
cd "$(dirname "$0")"
./vm-umount.sh || true
./vm-run.sh 'fshalt' >/dev/null 2>&1 || true   # the connection dies with the guest
sleep 25
virsh -c qemu:///session list --all
