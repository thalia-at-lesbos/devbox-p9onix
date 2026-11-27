# Development guide

How we write code in this environment. `AGENTS.md` covers operating the VM;
this file covers what to write and how.

Development happens **in the guest**. The host holds the source and the
tooling; 9front compiles and runs it.

## Languages and tools

Use the thing the platform already gives you. In order of preference:

| Job | Use | Not |
|---|---|---|
| Programs | **Go** (`go1.27.4 plan9/amd64`) | anything needing cgo — there is no cgo on plan9 |
| System code Go cannot reach | **Plan 9 C** (`6c`/`6l`, `libc.h`, `u.h`) | ISO C with POSIX assumptions |
| Scripts | **rc** | bash, sh, Python |
| Builds | **mk** with an `mkfile` | make, Makefiles, shell build scripts |

**Go is the default.** Write Go unless there is a reason not to. The standard
library works on plan9; `net`, `net/http`, `os` and `io` all behave.

**Plan 9 C where Go cannot go** — talking to a device file, writing a file
server, anything wanting the 9front libraries directly. Plan 9 C is not ANSI
C with extras: it has its own headers (`u.h` then `libc.h`, in that order),
its own compilers, and no `#include <stdio.h>`. `mk` drives it.

**rc for scripts.** Every script in the guest starts `#!/bin/rc`. rc is a
real shell with saner quoting than sh, and it is already there.

**mk for builds.** One `mkfile` per project. mk is simpler than make and
reads the same way as everything else in the system. A Go project still gets
an `mkfile` wrapping `go build`, so every project builds the same way.

A minimal `mkfile` for a Go program:

```
</$objtype/mkfile

TARG=hello

all:V:
	go build -o $TARG

clean:V:
	rm -f $TARG

install:V: all
	cp $TARG /$objtype/bin/$TARG
```

Targets ending in `:V:` are virtual — they are commands, not files.

## Project layout

One directory per project under `projects/`:

```
projects/<projectname>/
    readme.md      what it does, how to build it, how to run it
    mkfile         the build
    *.go           the source
```

`projects/` is the same directory on both machines: it is `./projects` on the
host and `/n/host` in the guest. Edit on either side; there is no sync step
and no copy to forget.

```sh
./tools/vm-run.sh 'cd /n/host/myproj; mk'
```

Every project gets a `readme.md`, in that order — **what it does, how to
build it, how to run it** — in plain prose. If you cannot say what it does in
two sentences, the project is doing too much.

## Design philosophy

Favour **simplicity and the Unix/Plan 9 philosophy above all else.**

- Small programs that do one thing, composed through pipes and files.
- Text streams as the universal interface. Prefer a line-oriented format you
  can `grep` over a structured one you need a library to read.
- Everything is a file; prefer the filesystem over bespoke IPC or daemons.
  If a program needs to expose state, consider serving a file system.
- No framework, dependency, or abstraction until the plain version is
  demonstrably insufficient. Deleting code is progress.
- Clarity over cleverness. If it needs a comment to explain *what* it does,
  rewrite it; comments explain *why*.
- Prefer the standard library. A Go project here should normally have no
  module requirements at all.

## Git practice

The host repository is ordinary git. Inside the guest, use **git9**, 9front's
native git, which spells subcommands as paths: `git/clone`, `git/commit`,
`git/log`, `git/diff`, `git/push`. There is no `git xxx` form.

Keep it simple and keep it honest:

- **Commit one logical change.** If the message needs the word "and", it is
  probably two commits.
- **Write the message for someone reading it in a year.** A short imperative
  summary line under ~50 characters, blank line, then *why* the change was
  needed. The diff already says what changed.
- **Commit working code.** Build it before committing; do not commit a tree
  you know is broken to "save progress".
- **Never commit build output or binaries.** Compiled programs, `*.qcow2`,
  ISOs and vendored upstream trees stay out. Check `git status` before you
  commit, not after.
- **Pull before you push**, and keep history readable rather than merging
  noise for its own sake.

In the guest:

```
git/log
git/diff
git/add main.go
git/commit -m 'serve index from memory' main.go
git/push
```

An example of a message worth writing:

```
serve index from memory

Reading index.html per request showed up as the top cost under load,
and the file never changes while the server is running.
```

## Gotchas

Hard-won; all of these have cost real time here.

### rc is not sh

Single quotes quote; **double quotes are ordinary characters**.
`aux/listen1 -t "tcp!*!17019" …` announces on a file literally called `"tcp`
and fails with a baffling error. A bare `=` is a syntax error too, so
`echo GOROOT=/x` must be `echo 'GOROOT=/x'`, and `echo ===` must be quoted.
This applies to everything passed to `tools/vm-run.sh`.

### `/usr/glenda/lib/profile` is a `switch($service)`

`case terminal` ends by running rio, so **anything appended to the end of the
file never runs**. Environment setup goes at the top; boot-time services go
in the terminal branch, before `rio -i riostart`.

### Do not background anything in `vm-run.sh`

A process that outlives the command keeps the rcpu connection open and the
script never returns. Long jobs run in the foreground where you can watch
them.

### Exit codes do not survive rcpu

A failing command gives a non-zero status, but not *its* number — `exit 3`
arrives as 114. Test for success, never for a particular code.

### Unmount `./mnt` before halting the VM

A FUSE mount whose server has gone away hangs every process that touches it,
including shell completion and your editor. `tools/vm-halt.sh` unmounts
first.

### Plan 9 `tar` cannot read GNU long names

Pack with `--format=ustar`; its prefix field covers paths up to 255
characters. Otherwise extraction fails on deep paths.

### No cgo

`GOOS=plan9` has no cgo. Any dependency reaching for C will not build. This
is a feature: it keeps dependencies honest.

### The console is the fallback, not the interface

Screenshots lag a beat behind the output and the window is ~18 lines. Use
`vm-run.sh`. Reach for `vm-key.sh` and `vm-shot.sh` only when rcpu is down —
and then read `AGENTS.md`, which documents the mouse and focus behaviour in
detail.
