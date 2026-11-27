# Installing devbox-p9onix on Ubuntu

Builds the host side, installs 9front from the ISO, and wires up the
host↔guest plumbing. End state is what `AGENTS.md` describes: a 9front guest
you drive with `tools/vm-run.sh`, with both file systems visible from both
sides.

Everything runs as your normal user under `qemu:///session`. The only step
needing root is installing packages.

Verified on Ubuntu with libvirt 12.0.0, QEMU 10.2.1 and virt-manager 5.1.0.

Throughout, `$REPO` is your clone of this repository and all commands run
from there unless stated otherwise.

## 1. Host packages

```sh
sudo apt update
sudo apt install -y \
    qemu-system-x86 qemu-utils \
    libvirt-daemon libvirt-daemon-driver-qemu libvirt-clients virt-manager \
    drawterm-9front \
    fuse3 libfuse3-dev meson ninja-build \
    build-essential git curl golang-go
```

What each group is for:

| Package | Why |
|---|---|
| `qemu-system-x86`, `qemu-utils` | the emulator and `qemu-img` |
| `libvirt-daemon*`, `libvirt-clients` | `virsh`, the session daemon |
| `virt-manager` | the GUI console, and `virt-xml` |
| `drawterm-9front` | rcpu client; `tools/vm-run.sh` is a wrapper round it |
| `fuse3`, `libfuse3-dev` | mounting the guest on `./mnt`; headers to build 9pfuse |
| `meson`, `ninja-build`, `build-essential` | building 9pfuse and u9fs |
| `golang-go` | bootstrap compiler for cross-building the guest's Go |
| `systemd` (already present) | `systemd-socket-activate` runs u9fs per connection |

## 2. KVM access

Without this the VM still runs, just slowly under emulation.

```sh
ls -l /dev/kvm            # must be readable and writable by you
sudo setfacl -m u:$USER:rw /dev/kvm
```

A group membership (`sudo usermod -aG kvm $USER`) works too, but only takes
effect in a **new login session** — an already-running shell keeps its old
groups, which is a confusing way to lose an afternoon.

## 3. libvirt session

`qemu:///session` needs no group and no root. It should answer straight away:

```sh
virsh -c qemu:///session list --all
```

The daemon is socket-activated, so there is nothing to start. If this errors,
fix it before going further.

> Do not use `qemu:///system` here. The domain XML, the port forwards and
> every script in `tools/` assume the session connection.

## 4. Build the 9P helpers

Two small programs, built in place and not committed to this repository.

**u9fs** — serves the host's `./projects` to the guest.

```sh
git clone https://github.com/Plan9-Archive/u9fs
cd u9fs && make && cd ..
```

Leaves `./u9fs/u9fs`, which `tools/vm-export.sh` runs directly.

**9pfuse** — mounts the guest's file system on the host.

```sh
git clone https://github.com/aperezdc/9pfuse
cd 9pfuse && meson setup build && meson compile -C build
sudo install -m755 build/9pfuse /usr/local/bin/9pfuse
cd ..
```

Check it is reachable and that you can use FUSE:

```sh
command -v 9pfuse fusermount3
ls -l /dev/fuse            # must be readable and writable by you
```

## 5. Fetch the 9front ISO

```sh
mkdir -p iso && cd iso
curl -LO http://9front.org/iso/9front-11952.amd64.iso.gz
gunzip 9front-11952.amd64.iso.gz
cd ..
```

Any later release should work, but the version this was built and tested
against is **11952** ("THIS WAS SUPPOSED TO BE FUN"). Decompress it — libvirt
will not boot the `.gz`.

## 6. Define the VM from the committed config

`9front.xml` is the working configuration: q35 + SeaBIOS, 2 vCPU, 2 GB, SATA
disk, e1000 NIC on a fixed PCI slot, std VGA, PS/2 input, and a
`<qemu:commandline>` block carrying the SLIRP port forwards.

Fix the disk path first — the XML has an absolute path baked in:

```sh
sed -i "s|<source file='.*/9front.qcow2'/>|<source file='$PWD/9front.qcow2'/>|" 9front.xml
qemu-img create -f qcow2 9front.qcow2 5G
virsh -c qemu:///session define 9front.xml
```

Then temporarily attach the ISO and boot from it, leaving everything else in
the config alone:

```sh
virt-xml -c qemu:///session 9front --add-device \
    --disk path=$PWD/iso/9front-11952.amd64.iso,device=cdrom,bus=sata,readonly=on
virt-xml -c qemu:///session 9front --edit --boot cdrom,hd
virsh -c qemu:///session start 9front
virt-manager --connect qemu:///session --show-domain-console 9front &
```

### Why BIOS and not UEFI

9front boots through the legacy BIOS path. virt-manager defaults new q35
guests to UEFI; the committed XML has no `<loader>`, which is what you want.
If you rebuild the domain by hand, set firmware to BIOS explicitly.

## 7. Run the installer

Click into the console window and work through it. Press Enter to accept a
default shown in `[brackets]`.

1. `bootargs` → Enter.
2. `user[glenda]` → Enter.
3. `vgasize` → type `text`. A text console is far easier to drive than rio
   during an install, and §8 sets the real resolution afterwards.
4. At `term%` run `inst/start`, then work the task list in the order offered:

| Task | Answer |
|---|---|
| `configfs` | `hjfs` — right choice for a small disk, no venti/worm to manage |
| `partdisk` | disk `sdE0`, table `mbr`, then `w` and `q` in fdisk |
| `prepdisk` | accept the default `/dev/sdE0/plan9`, then `w` and `q` |
| `mountfs` | accept `/dev/sdE0/fs`, accept the cache size, ream: `yes` |
| `confignet` | `automatic` (DHCP from SLIRP) |
| `mountdist` | accept the defaults |
| `copydist` | the long one; it ends with `file system made` |
| `ndbsetup` | accept sysname `cirno` |
| `tzsetup` | your zone |
| `bootsetup` | accept `/dev/sdE0/9fat`; install the Plan 9 MBR: `yes`; mark partition active: `yes` |
| `finish` | reboots |

Make sure `partdisk` targets **`sdE0`**, the hard disk. `sdE1` is the CD.

Then detach the ISO and boot from disk:

```sh
virsh -c qemu:///session destroy 9front
virt-xml -c qemu:///session 9front --remove-device --disk device=cdrom
virt-xml -c qemu:///session 9front --edit --boot hd
virsh -c qemu:///session start 9front
```

## 8. Graphics

A disk install writes `vgasize=text`, so rio cannot start and you land on a
bare `term%`. There is no `vgasize` prompt on a disk install — the value has
to come from `plan9.ini`. In the guest:

```
9fs 9fat
cp /n/9fat/plan9.ini /tmp/p.ini
```

Edit so it reads:

```
bootfile=9pc64
bootargs=local!/dev/sdE0/fs -m 296
mouseport=ps2
monitor=vesa
vgasize=1024x768x32
```

then `cp /tmp/p.ini /n/9fat/plan9.ini` and `fshalt`. On the next boot you get
rio.

Console driving at this stage is fiddly — see "the console" in `AGENTS.md`,
and `tools/vm-key.sh` / `tools/vm-focus.sh`, which exist for exactly this.

## 9. Cross-build Go for the guest

go.dev publishes no `plan9/amd64` binaries, so build the toolchain on the
host. `bootstrap.bash` produces a toolchain that *runs on* the target.

```sh
curl -LO https://go.dev/dl/go1.27.4.linux-amd64.tar.gz
tar xzf go1.27.4.linux-amd64.tar.gz          # -> ./go
cp -a go gosrc
cd gosrc/src
GOROOT_BOOTSTRAP=$PWD/../../go GOOS=plan9 GOARCH=amd64 ./bootstrap.bash
cd ../..
```

That leaves `./go-plan9-amd64-bootstrap/`. Confirm it really is Plan 9 code —
`file` will only say "data", so check the magic, which should be `0000 8a97`:

```sh
xxd -l 4 go-plan9-amd64-bootstrap/bin/go
```

Pack it as **ustar**, excluding test fixtures. Plan 9's `tar` cannot read GNU
long-name records; ustar's prefix field covers paths up to 255 characters,
which the tree fits inside once the fixtures are gone:

```sh
tar czf go-plan9.tar.gz --format=ustar \
    --exclude='*/test/*' --exclude='*/testdata' --exclude='*/testdata/*' \
    --exclude='*/api/*' --exclude='*/doc/*' --exclude='*/misc/*' \
    go-plan9-amd64-bootstrap
```

Serve it and pull it in from the guest, which reaches the host at `10.0.2.2`:

```sh
python3 -m http.server 8888 --bind 127.0.0.1 &
```

In the guest:

```
cd /sys/lib
hget http://10.0.2.2:8888/go-plan9.tar.gz > go.tar.gz
gunzip -c go.tar.gz | tar xf /fd/0
mv go-plan9-amd64-bootstrap go
rm go.tar.gz
```

The host-side trees (`go/`, `gosrc/`, `go-plan9-amd64-bootstrap/`, the
tarballs) are scratch; delete them once the guest has the toolchain.

## 10. Guest plumbing

Three pieces, all started from the `case terminal` branch of
`/usr/glenda/lib/profile` so they return on every boot.

Create `/rc/bin/hostfs`, which mounts the host's `projects` directory:

```
#!/bin/rc
# Mount the host's projects directory at /n/host.
unmount /n/host >[2]/dev/null
rm -f /srv/host
exec srv -n tcp!10.0.2.2!5640 host /n/host
```

Create `/rc/bin/service/tcp17019`, the rcpu service drawterm talks to:

```
#!/bin/rc
if(~ $#* 3){
	netdir=$3
	remote=$2!`{cat $3/remote}
}
fn server {
	~ $#remote 0 || echo -n $netdir $remote >/proc/$pid/args
	rm -f /env/'fn#server'
	. <{n=`{read} && ! ~ $#n 0 && read -c $n} >[2=1]
}
exec tlssrv -a /bin/rc -c server
```

`chmod +x` both.

Then edit `/usr/glenda/lib/profile`. Put the Go environment at the very
**top** of the file:

```
GOROOT=/sys/lib/go
GOPATH=/usr/glenda/go
bind -a /sys/lib/go/bin /bin
```

and the services in the **`case terminal`** branch, before `rio -i riostart`:

```
	echo 'key proto=dp9ik dom=9front user=glenda !password=glenda123' >/mnt/factotum/ctl
	aux/listen1 -t 'tcp!*!17019' /rc/bin/service/tcp17019 >/dev/null >[2=1] &
	aux/listen1 -t 'tcp!*!17007' /bin/exportfs -r / >/dev/null >[2=1] &
	hostfs >/dev/null >[2=1]
```

Placement is not cosmetic. The profile is a `switch($service)` whose
`case terminal` ends by running rio, so **anything appended to the end of the
file never runs**. Quote with single quotes only; in rc, double quotes are
ordinary characters, and a bare `=` is a syntax error.

The password is in cleartext by design for a loopback-only sandbox — read
"Authentication, and why it is this weak" in `AGENTS.md` before copying any
of this anywhere real.

## 11. Verify

Reboot the guest, then from the host:

```sh
./tools/vm-boot.sh                       # prints "9front ready"
./tools/vm-run.sh 'go version'           # go version go1.27.4 plan9/amd64
./tools/vm-run.sh 'ls /bin/git/clone'    # git9 ships in the base system
./tools/vm-run.sh 'ls /n/host'           # the host's ./projects
ls mnt/usr/glenda                        # the guest's files, from the host
```

A port-forward check, end to end:

```sh
./tools/vm-run.sh "aux/listen1 -t 'tcp!*!8000' /bin/echo hi" &
curl http://127.0.0.1:8000/
```

## 12. Cut a golden image

Once it all works, snapshot it. `vm-halt.sh` first — the image is worthless
if hjfs did not flush.

```sh
./tools/vm-halt.sh
qemu-img convert -O qcow2 -c -o compression_type=zstd 9front.qcow2 9front-golden2.qcow2
qemu-img check 9front-golden2.qcow2
qemu-img compare 9front.qcow2 9front-golden2.qcow2    # "Images are identical."
chmod 444 9front-golden2.qcow2
```

`tools/vm-restore.sh` rebuilds the working disk from it.

## Troubleshooting

**`virsh` says permission denied.** You are hitting `qemu:///system`. Pass
`-c qemu:///session`.

**VM will not start, "slot N function 0 not available".** Two devices want
the same PCI slot. The NIC is pinned to `bus=pcie.0,addr=0x5` in the XML;
pick another free slot if something else claims it.

**`rcpu never answered`.** The listeners live in the profile's terminal
branch. Check the console with `tools/vm-shot.sh`, and confirm the profile
edits are inside `case terminal` and before `rio`.

**Everything touching `./mnt` hangs.** The guest stopped serving while the
mount was up. `fusermount -uz mnt`. Always unmount before halting;
`vm-halt.sh` does.

**`tar` in the guest errors on long names.** The archive was not ustar.
Rebuild it with `--format=ustar` and the exclusions from §9.
