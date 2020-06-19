package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
)

const (
	medusaUA = "medusa/0.2.0"
	version  = "0.2.0"
)

const (
	urlTypeSingle = iota
	urlTypeMultiple

	defaultSchema = "http"
	schemaSlicer  = "://"
)

var (
	u  = flag.String("u", "", "")
	ul = flag.String("uL", "", "")

	ua = flag.String("ua", medusaUA, "")

	cp              = flag.String("cP", "200", "")
	cn              = flag.String("cN", "", "")
	e               = flag.String("e", "", "")
	s               = flag.String("s", "", "")
	insecure        = flag.Bool("x", false, "")
	followRedirects = flag.Bool("follow", false, "")
	recursive       = flag.Bool("r", false, "Enable recursive fuzzing")

	rcp = flag.String("rcP", "200", "")
	wl  = flag.String("w", "", "")

	cpus = flag.Int("cpus", runtime.GOMAXPROCS(-1), "")
)

//hit All found URLs are collected as hits.
type hit struct {
	m      sync.RWMutex
	url    string
	result map[string]string // [200]/info
}

type hits struct {
	m sync.RWMutex
	l map[string]hit
}

var usage = `Usage: medusa [options...]
Options:
-u                    Single URL  
-uL                   URL list file path (line by line)
-e                    Extension 
-s                    Force schema (uses default http if does not contains url)
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

var header = `
medusa v%s by rizasabuncu
-----------------------------------------
[*] URL/List: %s
[*] CPU Cores: %d
[*] User Agent: %s
[*] Extension: %s
[*] Positive status codes: %s
[*] Negative status codes: %s
[*] Bypass SSL verification %t
[*] Follow redirects: %t
[*] Recursive fuzz: %t
[*] Recursive positive status codes: %s
[*] Wordlist path: %s
[*] Wordlist length: %d
-----------------------------------------
`

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(usage, medusaUA, runtime.NumCPU()))
	}

	flag.Parse()
	if flag.NFlag() < 1 {
		usageAndExit("")
	}

	runtime.GOMAXPROCS(*cpus)

	if len(*u)+len(*ul) <= 0 {
		usageAndExit("-u or -uL cannot be empty")
	}

	if len(*wl) <= 0 {
		usageAndExit("-w cannot be empty")
	}

	var urlType = urlTypeSingle
	var urls = []string{}
	var wordlist = []string{}
	var err error
	var scanURL = *u

	if len(*ul) > len(*u) {
		urlType = urlTypeMultiple
		scanURL = *ul
	}

	switch urlType {
	case urlTypeSingle:
		urls = append(urls, scanURL)
		logInfo("URL loaded (" + *u + ")")
		break
	case urlTypeMultiple:

		_, err = os.Stat(scanURL)
		if err != nil {
			usageAndExit("URL list file not exists")
		}

		urls, err = readLines(scanURL)
		if err != nil {
			usageAndExit("URL list load error: " + err.Error())
		}

		break
	}

	_, err = os.Stat(*wl)
	if err != nil {
		usageAndExit("Wordlist file not exists")
	}

	wordlist, err = readLines(*wl)
	if err != nil {
		usageAndExit("Wordlist load error: " + err.Error())
	}

	//header
	fmt.Printf(
		header, version,
		scanURL, *cpus,
		*ua, *e,
		*cp, *cn,
		*insecure,
		*followRedirects,
		*recursive,
		*rcp,
		*wl, len(wordlist),
	)

	var h = &hits{}
	h.l = make(map[string]hit)

	// TODO
	//

	/* client := &http.Client{Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 5 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   -1,
	}} */

	for _, u := range urls {
		for _, w := range wordlist {
			var url = u

			if !strings.Contains(url, "http") || !strings.Contains(url, "https") {
				if len(*s) <= 0 {
					url = defaultSchema + schemaSlicer + url
				}
			}

			if len(*s) > 0 {
				if strings.Contains(url, "http") {
					strings.ReplaceAll(url, "http", *s)
				} else if strings.Contains(url, "https") {
					strings.ReplaceAll(url, "https", *s)
				} else {
					url = *s + schemaSlicer + url
				}
			}

			if url[len(u)-1:] != "/" {
				url = url + "/"
			}

			url = url + w
		}
	}
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func logInfo(msg string) {
	fmt.Fprintf(os.Stdin, "[+] "+msg+"\n")
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
