#!/bin/bash
# Run a command in the guest and read its output here.
#
#   ./tools/vm-run.sh 'go version'
#   echo 'ls /n/host' | ./tools/vm-run.sh
#   ./tools/vm-run.sh 'cat >/tmp/x' < file      # stdin is passed through
#
# The command is rc(1), not sh: quote with single quotes only, and quote any
# word containing '=' (`echo 'GOROOT=/x'`).
#
# The host's ./projects appears in the session as /mnt/term.
set -u
cd "$(dirname "$0")/.."
cmd="${1-}"
[ -n "$cmd" ] || cmd="$(cat)"
# Password of the throwaway sandbox VM; see AGENTS.md "Authentication".
exec env PASS=glenda123 drawterm -G -h 127.0.0.1 -a 127.0.0.1 -u glenda \
	-r "$PWD/projects" -c "$cmd"
