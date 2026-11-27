#!/bin/bash
# Serve ./projects to the guest over 9P (u9fs), so the guest can mount it.
# Listens on 127.0.0.1:5640; the guest reaches it as tcp!10.0.2.2!5640 and
# mounts it with `hostfs` (see /bin/hostfs in the guest).
# Port 5640 rather than the usual 564 because binding below 1024 needs root.
# Already running? Do nothing.
set -e
cd "$(dirname "$0")/.."
root="$PWD/projects"
log=/tmp/u9fs.log

if pgrep -f "systemd-socket-activate.*5640" >/dev/null; then
	echo "u9fs already serving $root on 127.0.0.1:5640"
	exit 0
fi

mkdir -p "$root"
exec systemd-socket-activate -a --inetd -l 127.0.0.1:5640 \
	"$PWD/u9fs/u9fs" -a none -u "$(id -un)" -l "$log" "$root"
