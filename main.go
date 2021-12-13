package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/pyke369/golang-support/multiflag"
)

var (
	PROGNAME = "mfetch"
	VERSION  = "1.1.0"

	Flagset     = flag.NewFlagSet(PROGNAME, flag.ExitOnError)
	Concurrency = 8
	Chunksize   = 8 << 20
	Timeout     = 10
	Retries     = 4
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
		fmt.Fprintf(os.Stderr, `usage:
  %s [<option...>] <argument...>

arguments:
  - client mode
        <source-url> [-|<local-file>|<target-url>]
  - server mode
        [<local-folder>]

options:
`, filepath.Base(os.Args[0]))
		Flagset.PrintDefaults()
	}
	Flagset.IntVar(&Concurrency, "concurrency", Concurrency, "set transfer concurrency level")
	Flagset.IntVar(&Chunksize, "chunksize", Chunksize, "set worker chunksize")
	Flagset.IntVar(&Timeout, "timeout", Timeout, "set requests timeout")
	Flagset.IntVar(&Retries, "retries", Retries, "set source request retries")
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
	Concurrency = int(math.Min(32, math.Max(1, float64(Concurrency))))
	Chunksize = int(math.Min(64<<20, math.Max(1<<20, float64(Chunksize))))
	Timeout = int(math.Min(30, math.Max(1, float64(Timeout))))
	Retries = int(math.Min(4, math.Max(0, float64(Retries))))
	Listen, Certificate, Password = strings.TrimLeft(strings.TrimSpace(Listen), "*"), strings.TrimSpace(Certificate), strings.TrimSpace(Password)
	if Listen == "" {
		Client()
	} else {
		Server()
	}
}
