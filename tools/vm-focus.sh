#!/bin/bash
# Put the guest pointer inside the rio terminal window and click to focus it.
# rio focus follows the mouse, and keystrokes are DISCARDED unless a window
# is focused. Run this once after boot, before sending any keys.
# Note: PS/2 is a relative mouse and the guest moves ~2x the requested delta,
# so we pin to the top-left corner first, then step in small increments
# (large deltas overflow the PS/2 packet and under-move).
set -e
M(){ virsh -c qemu:///session qemu-monitor-command 9front --hmp "$1" >/dev/null; }
for i in $(seq 1 10); do M "mouse_move -100 -100"; done   # pin to (0,0)
for i in $(seq 1 11); do M "mouse_move 15 13"; done       # -> approx (330,286)
M "mouse_button 1"; sleep 0.3; M "mouse_button 0"         # click to focus
