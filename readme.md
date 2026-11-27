# devbox-p9onix

A simple agentic development environment for Plan 9 on a Linux host.

**devbox-p9onix** — the virtual **dev**elopment **box** for **p9** **on**
Lin**ix** — is a 9front virtual machine, the scripts that drive it, and the
notes an agent or a person needs to work in it. The host keeps the source and
the tooling; the guest compiles and runs the code.

The point is that the VM is *drivable*. Rather than a console you have to
watch, the host gets a command channel and a shared file system:

```sh
./tools/vm-boot.sh                    # boot the guest and bring up the plumbing
./tools/vm-run.sh 'go version'        # run a command in the guest, read its output
ls mnt/usr/glenda                     # the guest's files, from the host
./tools/vm-halt.sh                    # clean shutdown
```

The guest sees the host's `projects/` directory at `/n/host`, and the host
sees the guest's root at `./mnt`. Same files, both directions, no copying.

## What is in the box

- 9front 11952 (amd64), hjfs on a 5 GiB disk, rio at 1024x768
- Go 1.27.4 for `plan9/amd64`, cross-built on the host
- git9, 9front's native git, from the base system
- rcpu over drawterm for commands, 9P both ways for files
- A read-only, compressed golden image to reset to

Everything runs as a normal user under `qemu:///session`. No root, no system
libvirt, no bridge.

## Documentation

| | |
|---|---|
| [`docs/install.md`](docs/install.md) | set it up on Ubuntu from scratch |
| [`docs/dev.md`](docs/dev.md) | how to write code here — Go, rc, mk, and the gotchas |
| [`AGENTS.md`](AGENTS.md) | operating the VM: tooling, networking, recovery |

## Layout

```
projects/<name>/   development source, one dir per project
tools/             host-side scripts for driving the VM
docs/              setup and development guides
9front.xml         libvirt domain definition
```

The VM disk images, the install ISO, and the upstream `u9fs` and `9pfuse`
trees are build inputs, not source; they are not in this repository.
`docs/install.md` says how to obtain and build each one.

## A warning

This VM is a **loopback-only sandbox** with deliberately weak authentication:
a cleartext password in the scripts, and a 9P export with no authentication
at all. Anyone who can open a socket on the host has full access to the
guest. Keep the forwarded ports bound to `127.0.0.1`, keep nothing secret in
the VM, and do not copy this arrangement onto a machine other people can
reach. See "Authentication, and why it is this weak" in `AGENTS.md`.
