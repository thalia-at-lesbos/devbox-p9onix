// Test serves the status of the 9front system it runs on as a web page.
//
// Every number comes from a file the kernel already keeps: /dev/time,
// /dev/swap, /dev/sysstat and /net/ipifc. Nothing is cached, so a page is
// true at the moment it is read and stale a moment later.
package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type row struct{ Key, Val string }

type section struct {
	Name string
	Rows []row
}

func main() {
	addr := flag.String("a", ":8000", "serve on this address")
	flag.Parse()

	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/status.txt", serveText)
	log.Print("serving on ", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := page.Execute(w, struct {
		Host     string
		Now      string
		Sections []section
	}{
		Host:     read("/dev/sysname"),
		Now:      now(),
		Sections: status(),
	})
	if err != nil {
		log.Print(err)
	}
}

// serveText writes the same numbers as section, key, value: one fact a line,
// so the page is as useful to grep as to a browser.
func serveText(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	for _, s := range status() {
		for _, r := range s.Rows {
			fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, r.Key, r.Val)
		}
	}
}

func status() []section {
	return []section{
		{"system", system()},
		{"memory", memory()},
		{"processors", processors()},
		{"network", network()},
	}
}

func system() []row {
	return []row{
		{"time", now()},
		{"uptime", uptime()},
		{"sysname", read("/dev/sysname")},
		{"osversion", read("/dev/osversion")},
		{"kernel", strings.TrimLeft(firstLine("/dev/config"), "# ")},
		{"arch", runtime.GOOS + "/" + runtime.GOARCH},
		{"user", read("/dev/user")},
		{"hostowner", read("/dev/hostowner")},
		{"processes", strconv.Itoa(len(dirnames("/proc")))},
	}
}

func now() string {
	return time.Now().Format("Mon 2 Jan 2006 15:04:05 MST")
}

// uptime divides the clock ticks since boot by the clock frequency, the
// third and fourth numbers in /dev/time.
func uptime() string {
	f := strings.Fields(read("/dev/time"))
	if len(f) < 4 {
		return "unknown"
	}
	ticks, err1 := strconv.ParseFloat(f[2], 64)
	freq, err2 := strconv.ParseFloat(f[3], 64)
	if err1 != nil || err2 != nil || freq <= 0 {
		return "unknown"
	}
	s := int64(ticks / freq)
	d, s := s/86400, s%86400
	h, m, s := s/3600, s%3600/60, s%60
	switch {
	case d > 0:
		return fmt.Sprintf("%dd %dh %dm %ds", d, h, m, s)
	case h > 0:
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	default:
		return fmt.Sprintf("%dm %ds", m, s)
	}
}

// memory reads /dev/swap, whose every line is a value and the name of what
// it counts; the counts of pages are written used/total.
func memory() []row {
	swap := map[string]string{}
	for _, l := range strings.Split(read("/dev/swap"), "\n") {
		if f := strings.Fields(l); len(f) >= 2 {
			swap[strings.Join(f[1:], " ")] = f[0]
		}
	}
	size := number(swap["pagesize"])
	return []row{
		{"total", bytes(number(swap["memory"]))},
		{"page size", bytes(size)},
		{"kernel", bytes(number(swap["kernel"]) * size)},
		{"user", pages(swap["user"], size)},
		{"swap", pages(swap["swap"], size)},
	}
}

func pages(used string, size int64) string {
	a, b, ok := strings.Cut(used, "/")
	if !ok {
		return "unknown"
	}
	have, total := number(a), number(b)
	if total == 0 {
		return "none"
	}
	return fmt.Sprintf("%s of %s (%d%%)", bytes(have*size), bytes(total*size), 100*have/total)
}

func bytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

// processors reads /dev/sysstat, one line a processor, ten numbers a line:
// number, context switches, interrupts, system calls, page faults, TLB
// faults, TLB purges, load average in milli-CPUs, idle and interrupt time
// as percentages.
func processors() []row {
	var rows []row
	for _, l := range strings.Split(read("/dev/sysstat"), "\n") {
		f := strings.Fields(l)
		if len(f) < 10 {
			continue
		}
		cpu := "cpu " + f[0] + " "
		rows = append(rows,
			row{cpu + "load", fmt.Sprintf("%.2f", float64(number(f[7]))/1000)},
			row{cpu + "idle", f[8] + "%"},
			row{cpu + "interrupt", f[9] + "%"},
			row{cpu + "context switches", f[1]},
			row{cpu + "interrupts", f[2]},
			row{cpu + "system calls", f[3]},
			row{cpu + "page faults", f[4]},
		)
	}
	return rows
}

// network reads each interface's status: a first line of a device name and
// then key value pairs, and a line for every address bound to it.
func network() []row {
	dirs, _ := filepath.Glob("/net/ipifc/[0-9]*")
	var rows []row
	for _, dir := range dirs {
		lines := strings.Split(read(path.Join(dir, "status")), "\n")
		f := strings.Fields(lines[0])
		if len(f) < 2 {
			continue
		}
		stat := map[string]string{}
		for i := 2; i+1 < len(f); i += 2 {
			stat[f[i]] = f[i+1]
		}
		name := "ipifc " + path.Base(dir) + " "
		rows = append(rows, row{name + "device", f[1]})
		for _, l := range lines[1:] {
			if a := strings.Fields(l); len(a) >= 2 {
				rows = append(rows, row{name + "address", a[0] + a[1]})
			}
		}
		rows = append(rows,
			row{name + "packets", "in " + stat["pktin"] + ", out " + stat["pktout"]},
			row{name + "errors", "in " + stat["errin"] + ", out " + stat["errout"]},
			row{name + "maximum transfer unit", stat["maxtu"]},
		)
	}
	return rows
}

func read(file string) string {
	b, err := os.ReadFile(file)
	if err != nil {
		log.Print(err)
		return ""
	}
	return strings.TrimRight(string(b), " \n")
}

func firstLine(file string) string {
	l, _, _ := strings.Cut(read(file), "\n")
	return l
}

func dirnames(dir string) []string {
	d, err := os.Open(dir)
	if err != nil {
		log.Print(err)
		return nil
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		log.Print(err)
	}
	return names
}

func number(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

var page = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="refresh" content="5">
<title>{{.Host}} status</title>
<style>
body { margin: 0; padding: 2rem 1rem; background: #ffffea; color: #222;
	font: 14px/1.5 "DejaVu Sans Mono", monospace; }
main { max-width: 42rem; margin: 0 auto; }
h1 { margin: 0; font-size: 1.4rem; }
h2 { margin: 2rem 0 .5rem; font-size: 1rem; font-weight: normal;
	color: #666; border-bottom: 1px solid #ddd9c0; padding-bottom: .25rem; }
p { margin: .25rem 0 0; color: #666; }
dl { display: grid; grid-template-columns: 14rem 1fr; gap: .15rem 1rem; margin: 0; }
dt { color: #666; }
dd { margin: 0; overflow-wrap: anywhere; }
a { color: #005577; }
@media (max-width: 30rem) { dl { grid-template-columns: 1fr; gap: 0 0; }
	dd { margin-bottom: .5rem; } }
</style>
</head>
<body>
<main>
<h1>{{.Host}}</h1>
<p>{{.Now}}</p>
{{range .Sections}}<h2>{{.Name}}</h2>
<dl>
{{range .Rows}}<dt>{{.Key}}</dt><dd>{{.Val}}</dd>
{{end}}</dl>
{{end}}<h2>also</h2>
<p><a href="/status.txt">status.txt</a> — the same numbers, one a line, for grep.</p>
</main>
</body>
</html>
`))
