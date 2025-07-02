package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/pyke369/golang-support/multiflag"
)

const (
	PROGNAME = "mfetch"
	PROGVER  = "1.3.0"
)

var (
	Flagset     = flag.NewFlagSet(PROGNAME, flag.ExitOnError)
	Version     = false
	Concurrency = 6
	Maxmem      = 6 * 64 << 20
	Timeout     = 10
	Source      = multiflag.Multiflag{}
	Target      = multiflag.Multiflag{}
	Post        = false
	Insecure    = false
	Noresume    = false
	Verbose     = false
	Dump        = false
	Progress    = false
	Listen      = ""
	Certificate = ""
	Password    = ""
)

func main() {
	Flagset.Usage = func() {
		os.Stderr.WriteString(strings.Join([]string{
			"usage:",
			"  " + filepath.Base(os.Args[0]) + " [<option...>] <argument...>",
			"",
			"arguments:",
			"  - client mode",
			"        <source-url> [-|<local-file>|<target-url>]",
			"  - server mode",
			"        [<local-folder>]",
			"",
			"options:",
			"",
		}, "\n"))
		Flagset.PrintDefaults()
	}
	Flagset.BoolVar(&Version, "version", Version, "show program version and exit")
	Flagset.IntVar(&Concurrency, "concurrency", Concurrency, "set transfer concurrency level")
	Flagset.IntVar(&Maxmem, "maxmem", Maxmem, "set maximum memory used for in-memory transfers")
	Flagset.IntVar(&Timeout, "timeout", Timeout, "set requests timeout")
	Flagset.Var(&Source, "source", "add HTTP header to source request (repeatable, no default)")
	Flagset.Var(&Target, "target", "add HTTP header to target request (repeatable, no default)")
	Flagset.BoolVar(&Post, "post", Post, "use HTTP POST method for remote target (default PUT)")
	Flagset.BoolVar(&Insecure, "insecure", Insecure, "ignore remote TLS certificate errors (default false)")
	Flagset.BoolVar(&Noresume, "noresume", Noresume, "disable transfer auto-resuming (default false)")
	Flagset.BoolVar(&Verbose, "verbose", Verbose, "set verbose mode (default false)")
	Flagset.BoolVar(&Dump, "dump", Dump, "dump HTTP requests and responses (default false)")
	Flagset.BoolVar(&Progress, "progress", Progress, "emit transfer progress JSON indications (default false)")
	Flagset.StringVar(&Listen, "listen", Listen, "set listening address & port in server mode (default client mode)")
	Flagset.StringVar(&Certificate, "certificate", Certificate, `use provided TLS certificate & key in server mode (or "internal", no default)`)
	Flagset.StringVar(&Password, "password", Password, "set security password in server mode (no default)")
	Flagset.Parse(os.Args[1:])
	Concurrency = min(32, max(1, Concurrency))
	Maxmem = (max(Concurrency*8<<20, Maxmem) / Concurrency) * Concurrency
	Timeout = min(30, max(1, Timeout))
	Listen, Certificate, Password = strings.TrimLeft(strings.TrimSpace(Listen), "*"), strings.TrimSpace(Certificate), strings.TrimSpace(Password)

	if Version {
		os.Stdout.WriteString(PROGNAME + " v" + PROGVER + "\n")
		return
	}

	if Listen != "" {
		Server()
		return
	}
	Client()
}
