#!/bin/bash
# Start the VM, walk it through both boot prompts, and bring up the plumbing:
# u9fs serving ./projects to the guest, and the guest's file system on ./mnt.
set -e
cd "$(dirname "$0")"

pgrep -f "systemd-socket-activate.*5640" >/dev/null ||
	setsid ./vm-export.sh >/tmp/u9fs-serve.log 2>&1 &

virsh -c qemu:///session start 9front 2>/dev/null || echo "(already running)"
sleep 22
./vm-key.sh enter    # bootargs: local!/dev/sdE0/fs -m 296
sleep 10
./vm-key.sh enter    # user[glenda]
sleep 20
./vm-focus.sh        # so the console is usable as a fallback

# The profile starts the listeners last; wait for rcpu to answer.
for i in $(seq 1 45); do
	if ./vm-run.sh 'exit' >/dev/null 2>&1; then
		./vm-mount.sh
		echo "9front ready: ./tools/vm-run.sh for commands, ./mnt for files"
		exit 0
	fi
	sleep 2
done
echo "rcpu never answered; check the console with ./tools/vm-shot.sh" >&2
exit 1
