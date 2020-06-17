package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

const (
	medusaUA = "medusa/0.2.0"
)

var (
	u  = flag.String("u", "", "")
	ul = flag.String("uL", "", "")

	ua = flag.String("ua", medusaUA, "")

	cp = flag.String("cP", "", "")
	cn = flag.String("cN", "", "")
	e  = flag.String("e", "", "")

	insecure        = flag.Bool("x", false, "")
	followRedirects = flag.Bool("follow", false, "")
	recursive       = flag.Bool("r", false, "Enable recursive fuzzing")

	rcP = flag.String("rcP", "200", "")
	wl  = flag.String("w", "", "")

	cpus = flag.Int("cpus", runtime.GOMAXPROCS(-1), "")
)

var usage = `Usage: medusa [options...]
Options:
  -u                    Single URL  
  -uL                   URL list file path (line by line)
  -e                    Extension 
  -ua                   User-agent value (default %s)
  -cP                   Postive status codes (seperate by comma)
  -cN                   Negative status codes (seperate by comma)
  -x                    Bypass SSL verification
  -r                    Follow redirects
  -rcP                  Positive status codes for recursive fuzzing (seperate by comma)
  -w                    Directory wordlist (line by line)
  -cpus                 Number of used cpu cores.
                        (default for current machine is %d cores)
`

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(usage, medusaUA, runtime.NumCPU()))
	}

	flag.Parse()
	if flag.NArg() < 1 {
		usageAndExit("")
	}

	runtime.GOMAXPROCS(*cpus)

	if len(*u) <= 0 || len(*ul) <= 0 {
		usageAndExit("-u or -uL cannot be empty")
	}

	if len(*wl) <= 0 {
		usageAndExit("-w cannot be empty")
	}

}

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}
