#!/bin/bash
# Mount the guest's file system on the host at ./mnt, over the guest's
# exportfs listener (port 17007) via 9pfuse.
set -e
cd "$(dirname "$0")/.."
if grep -q " $PWD/mnt fuse" /proc/mounts; then
	echo "already mounted on $PWD/mnt"
	exit 0
fi
mkdir -p mnt
9pfuse 'tcp!127.0.0.1!17007' mnt 2>>/tmp/9pfuse.log
echo "guest root mounted on $PWD/mnt"
