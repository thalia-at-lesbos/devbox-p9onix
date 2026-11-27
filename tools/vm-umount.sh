#!/bin/bash
# Unmount the guest file system from ./mnt.  Always do this before halting
# the VM: the mount goes stale when the guest stops serving it.
set -e
cd "$(dirname "$0")/.."
grep -q " $PWD/mnt fuse" /proc/mounts || { echo "not mounted"; exit 0; }
fusermount -uz mnt
echo "unmounted $PWD/mnt"
