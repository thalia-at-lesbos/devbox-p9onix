# test

A web page of this 9front system's status, served from the guest to the host.

Every number is read, at the moment of the request, from a file the kernel
already keeps: the date and time and the uptime from `/dev/time`, the machine
and the user from `/dev/sysname`, `/dev/osversion`, `/dev/config`, `/dev/user`
and `/dev/hostowner`, the process count from `/proc`, memory from `/dev/swap`,
per-processor load and counters from `/dev/sysstat`, and the interfaces and
their packet counts from `/net/ipifc/*/status`. Nothing is cached and nothing
is stored, so the page is true when it is read and stale a moment later; it
refreshes itself every five seconds.

Two paths: `/` is the page, and `/status.txt` is the same facts as lines of
section, key and value separated by tabs, for `grep` and `awk`.

## Building

In the guest, where Go lives:

	./tools/vm-run.sh "cd /n/host/test; go build '-buildvcs=false'"

The quotes matter — rc reads a bare `=` as a syntax error. The flag is needed
because Go looks for a version-control directory above the source and misreads
what it finds over 9P; the build itself needs nothing but the standard library.

## Running

	./tools/vm-run.sh 'cd /n/host/test; ./test' &

Then open <http://127.0.0.1:8000/> on the host, or `curl
http://127.0.0.1:8000/status.txt`. The server listens on every address inside
the guest, and QEMU forwards the host's `127.0.0.1:8000` to it.

Keep the server in the foreground of `vm-run.sh` and background the host side,
as above: a process left running inside `vm-run.sh` holds the rcpu connection
open and the script never returns. Stopping the host side does not always
reach the server, so if the port stays busy, `./tools/vm-run.sh 'kill test |
rc'` clears it.

`-a` takes another address, e.g. `./test -a :8001`; ports 8000 to 8010 are the
ones forwarded to the host.
