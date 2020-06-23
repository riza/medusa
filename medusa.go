package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
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
	recursive       = flag.Bool("r", false, "")

	rcp = flag.String("rcP", "200", "")
	wl  = flag.String("w", "", "")

	conc = flag.Int("conc", 100, "")

	cpus = flag.Int("cpus", runtime.GOMAXPROCS(-1), "")
)

//hit All found URLs are collected as hits.
type hit struct {
	url    string
	status int
}

type hits struct {
	m sync.RWMutex
	l map[string]hit
}

type response struct {
	*http.Response
	err error
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
-follow               Follow redirects
-r                    Enable recursive fuzzing
-rcP                  Positive status codes for recursive fuzzing (seperate by comma)
-w                    Directory wordlist (line by line)
-conc                 Maximum concurrent requests
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

	/* var scanl = &scan{}
	scanl.l = make(map[string][]string) */

	var hits = &hits{}
	hits.l = make(map[string]hit)

	reqChan := make(chan *http.Request)
	respChan := make(chan response)

	totalReq := len(urls) * len(wordlist)

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
			go dispatcher(url, reqChan)
		}
	}

	start := time.Now()

	go workerPool(*conc, reqChan, respChan)

	conns, size := consumer(totalReq, respChan)
	if conns >= int64(totalReq) {
		close(reqChan)
	}

	took := time.Since(start)
	ns := took.Nanoseconds()
	av := ns / conns

	average, err := time.ParseDuration(fmt.Sprintf("%d", av) + "ns")
	if err != nil {
		log.Println(err)
	}

	fmt.Printf("Connections:\t%d\nConcurrent:\t%d\nTotal size:\t%d bytes\nTotal time:\t%s\nAverage time:\t%s\n", conns, conc, size, took, average)
}

func dispatcher(url string, reqChan chan *http.Request) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Print(url)
	}
	reqChan <- req
}

func workerPool(max int, reqChan chan *http.Request, respChan chan response) {
	t := &http.Transport{
		DisableKeepAlives: true,
	}
	for i := 0; i < max; i++ {
		go worker(t, reqChan, respChan)
	}
}

func worker(t *http.Transport, reqChan chan *http.Request, respChan chan response) {
	for req := range reqChan {
		resp, err := t.RoundTrip(req)
		r := response{resp, err}
		respChan <- r
	}
}

func consumer(reqs int, respChan chan response) (int64, int64) {
	var (
		conns int64
		size  int64
	)
	for conns < int64(reqs) {
		select {
		case r, ok := <-respChan:
			if ok {
				if r.err != nil {
					log.Print(r.Request.URL)
				} else {
					size += r.ContentLength
					if err := r.Body.Close(); err != nil {
						log.Println(r.err)
					}

					if r.StatusCode == 200 {
						fmt.Println(r.Request.URL)
					}

				}
				conns++
			}
		}
	}
	return conns, size
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
