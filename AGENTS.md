# 9front development environment

A 9front VM is the development environment for this repository. The host runs
Linux and holds the source; **code is built and run inside the VM**.

The host drives the VM over 9P and rcpu: you can run commands in the guest and
read their output, and both machines can see each other's files. The emulated
keyboard and framebuffer are still there, but only as a fallback.

## Layout

```
./projects/<projectname>/   all development source, one dir per project,
                            each with its own readme.md
./tools/                    host-side scripts for driving the VM
./docs/                     install.md (host setup) and dev.md (writing code)
./mnt/                      the guest's file system, while mounted (9pfuse)
./9pfuse/                   9P-over-FUSE client, built; installed in /usr/local/bin
./u9fs/                     9P server for Unix, built in place as ./u9fs/u9fs
./9front.qcow2              live VM disk (writable, ~862 MB)
./9front-golden2.qcow2      read-only golden image, current (zstd, 353 MB)
./9front-golden.qcow2       read-only golden image, before the plumbing
./9front.xml                libvirt domain definition
./iso/                      9front install ISO
```

## Writing code

This file is about **operating** the VM. What to write and how lives in
[`docs/dev.md`](docs/dev.md): language choices (Go, Plan 9 C, rc, mk), project
layout, the design philosophy, git practice, and the gotchas that will cost
you an afternoon. Read it before writing anything in the guest.

Setting the whole thing up from scratch is [`docs/install.md`](docs/install.md).

## VM facts

| | |
|---|---|
| OS | 9front 11952 "THIS WAS SUPPOSED TO BE FUN", amd64 |
| Filesystem | hjfs on `/dev/sdE0/fs`, 5 GiB disk (MBR, BIOS boot) |
| Machine | q35 + SeaBIOS, 2 vCPU, 2 GB RAM, KVM accelerated |
| Devices | SATA disk, e1000 NIC, std VGA, PS/2 mouse+keyboard |
| Display | rio at 1024x768x32 (`vgasize` in `/n/9fat/plan9.ini`) |
| User | `glenda`, hostname `cirno` |
| libvirt | `qemu:///session` (user session — **not** `qemu:///system`) |

Installed toolchain:

- **Go 1.27.4 plan9/amd64** at `/sys/lib/go`, cross-built on the host.
  `GOROOT`, `GOPATH` and a `bind` of `/sys/lib/go/bin` into `/bin` are set at
  the top of `/usr/glenda/lib/profile`. Just run `go`.
- **git9** — 9front's native git, in the base system at `/bin/git/*`
  (`git/clone`, `git/commit`, `git/log`, `git/pull`, …). Note the `git/xxx`
  subcommand-as-path form; there is no `git xxx` command.

## Driving the VM

```sh
./tools/vm-boot.sh                  # start the VM and all the plumbing
./tools/vm-run.sh 'go version'      # run a command in the guest, read its output
./tools/vm-halt.sh                  # clean shutdown (fshalt)
```

`vm-boot.sh` is the only thing that needs the keyboard: it answers the two
boot prompts, then waits for the guest's rcpu listener and mounts the guest on
`./mnt`. It takes about 90 seconds and prints `9front ready` when the guest is
actually reachable — it does not guess with `sleep`.

`vm-run.sh` runs a command through `drawterm -G` over rcpu. Stdin, stdout and
stderr are all connected, so the guest composes with host pipelines:

```sh
./tools/vm-run.sh 'ls -l /n/host'
echo 'cd /n/host/foo; go build' | ./tools/vm-run.sh
./tools/vm-run.sh 'cat >/tmp/x' < localfile
./tools/vm-run.sh 'cat /sys/log/listen' | grep -i error
```

Two things to know before you use it: **do not background anything in
`vm-run.sh`**, or the script never returns, and exit *codes* do not survive
rcpu, so test for success rather than a particular number. Both are explained
in [`docs/dev.md`](docs/dev.md#gotchas), along with rc's quoting rules, which
apply to every command you pass in.

## Moving files

Three views of the same two file systems. Pick whichever is closest to hand;
they are all live, not copies.

| From | Path | Served by |
|---|---|---|
| host → sees guest `/` | `./mnt/` | guest `exportfs` + host `9pfuse` |
| guest → sees host `./projects` | `/n/host` | host `u9fs`, mounted at boot |
| guest → sees host `./projects` | `/mnt/term` | `drawterm -r`, per session |

So on the host, `cp`, `grep`, `$EDITOR` and friends work on the guest's files:

```sh
grep -rn 'func main' mnt/usr/glenda/
cp mnt/sys/log/listen /tmp/
```

and in the guest, the host's source tree is just a directory:

```sh
./tools/vm-run.sh 'cd /n/host/myproj; go build'
```

`/n/host` is mounted by `/rc/bin/hostfs` in the guest, from the terminal branch
of `/usr/glenda/lib/profile`, and posted to `/srv/host` so every window and
every rcpu session can mount it. Re-run `hostfs` in the guest if the host's
`u9fs` was restarted.

Unmount `./mnt` **before** halting the VM (`vm-halt.sh` does this); a mount
whose server has gone away hangs every process that touches it.

## How the plumbing fits together

```
host                                            guest (10.0.2.15)
----                                            -----------------
drawterm -G ----------- tcp 17019 (rcpu) -----> aux/listen1 + tlssrv -> rc
9pfuse ./mnt ---------- tcp 17007 (9P) -------> aux/listen1 + exportfs -r /
u9fs ./projects <------ tcp 5640 (9P) --------- srv -> /srv/host -> /n/host
```

The two listeners in the guest are started from the `case terminal` branch of
`/usr/glenda/lib/profile`, so they come back on every boot. The host's `u9fs`
is started by `vm-boot.sh` (or `./tools/vm-export.sh` on its own) under
`systemd-socket-activate`, which is the inetd this needs and wants no root.

## Networking

User-mode (SLIRP) networking, so addresses are private to the VM:

- Guest IP: **`10.0.2.15`**
- Host, as seen from the guest: **`10.0.2.2`**
- The host **cannot** reach `10.0.2.15` directly. Host→guest works only
  through forwarded ports. Guest→host needs no setup at all.

**Forwarded ports** (host side bound to `127.0.0.1` only):

| Port | Use |
|---|---|
| 8000–8010 | yours; a server on `10.0.2.15:8000` is `http://127.0.0.1:8000/` |
| 17019 | rcpu, for `drawterm` |
| 17007 | the guest's `exportfs`, for `9pfuse` |

The host's `u9fs` listens on `127.0.0.1:5640` — 5640 rather than the usual 564
because binding below 1024 needs root. No forwarding is involved; the guest
dials out to `tcp!10.0.2.2!5640`.

To forward more ports, edit the `hostfwd=` list in the `<qemu:commandline>`
block of `9front.xml`, redefine with `virsh -c qemu:///session define
9front.xml`, and restart the VM.

## Authentication, and why it is this weak

`glenda`'s 9front password is **`glenda123`**, in cleartext in
`tools/vm-run.sh` and in `/usr/glenda/lib/profile` in the guest.

The guest is its own authority: the profile loads a `dp9ik` key into
`factotum`, and rcpu authenticates the hostowner against it. There is no auth
server, no nvram key, and no certificate — `tlssrv -a` derives the TLS session
key from the p9any exchange.

**This is dangerous, and acceptable only because this VM is a loopback-only
sandbox.** Everything that could be abused is bound to `127.0.0.1`, and the
`exportfs` on 17007 has *no authentication at all* — anyone who can open a
socket on this host has full read/write access to the guest's root file
system, as glenda. Do not forward these ports to another address, do not put
anything secret in this VM, and do not copy this arrangement to a machine that
other people can reach.

## Golden images

Read-only (mode 444), zstd-compressed snapshots of a known-good system,
verified with `qemu-img check` and byte-identical to their source disk via
`qemu-img compare`.

- `9front-golden2.qcow2` — current: 9front + Go + git9 + the host plumbing.
- `9front-golden.qcow2` — the same system before the plumbing, kept as a way
  back if a change to the profile or the listeners ever makes the VM
  unreachable.

```sh
./tools/vm-restore.sh      # restore from golden2 (destroys current state)
./tools/vm-restore.sh 1    # restore from the original golden image
```

To cut a new one: `vm-halt.sh`, then

```sh
qemu-img convert -O qcow2 -c -o compression_type=zstd 9front.qcow2 new.qcow2
qemu-img check new.qcow2 && qemu-img compare 9front.qcow2 new.qcow2
chmod 444 new.qcow2
```

## The console, for when the plumbing is broken

If rcpu stops answering, the emulated keyboard and framebuffer are still
there. Read output by taking a screenshot and **looking at the image**.

```sh
./tools/vm-focus.sh                 # pin the pointer into the rio window
./tools/vm-key.sh line "echo hi"    # type a command and press Enter
./tools/vm-key.sh raw KEY_DELETE    # interrupt (DEL, not Ctrl-C)
./tools/vm-shot.sh /tmp/out.png     # capture the screen (PNG), then view it
```

### Console mechanics

The language-level traps — rc quoting, the profile's `switch`, backgrounding,
exit codes, `tar` — are in [`docs/dev.md`](docs/dev.md#gotchas). These are the
ones specific to driving the console by hand.

1. **Focus rio before typing.** rio's keyboard focus follows the mouse, and
   keystrokes sent while the pointer is over the desktop are silently
   discarded. `vm-focus.sh` pins the pointer to the corner, steps it into the
   terminal, and clicks.
   - The PS/2 mouse is *relative*, and the guest moves ~**2x** the requested
     delta. Large deltas overflow the PS/2 packet and under-move, so move in
     small increments from a known corner. Do not add a USB tablet — 9front
     tracked it unpredictably and the pointer became unrecoverable.
   - QEMU's `mouse_button` mask is **1 left, 2 right, 4 middle**. `2` gets
     rio's New/Resize/Move/Delete/Hide menu; hold it, release over `New`,
     then sweep a rectangle with it held again to get a fresh window. That is
     the way out of a window that has wedged.

2. **Booting needs two Enters**: one at `bootargs is (...)`, one at
   `user[glenda]:`. `vm-boot.sh` does both.

3. **Shut down with `fshalt`**, never `virsh destroy`, or hjfs may be damaged.

4. **Never run `con` on the console.** It puts the window in raw mode, so Del
   no longer interrupts and typed text is swallowed; the window is then only
   recoverable through rio's menu. Use `vm-run.sh` instead.

5. The terminal window is small (~18 lines) and its screenshots lag a beat
   behind the output. Prefer `vm-run.sh`, which has none of these problems.

## Host requirements

Full setup instructions, including how to build this from a bare Ubuntu
install, are in [`docs/install.md`](docs/install.md).

- `qemu-system-x86_64`, `virt-manager`, `virsh`, `qemu-img`
- `drawterm-9front` (`/usr/bin/drawterm`) — rcpu client
- `9pfuse` (`/usr/local/bin/9pfuse`, built from `./9pfuse`) and `fusermount`,
  with `/dev/fuse` readable/writable by the user
- `systemd-socket-activate` (systemd), to run `u9fs` per connection
- `/dev/kvm` readable/writable by the user (granted here via ACL)
- No root needed: everything runs under `qemu:///session`
- Open the GUI with:
  `virt-manager --connect qemu:///session --show-domain-console 9front`
