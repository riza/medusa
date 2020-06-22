package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
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
	url    string
	status int
}

type hits struct {
	m sync.RWMutex
	l map[string]hit
}

type scan struct {
	m sync.RWMutex
	l map[string][]string
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
var wg sync.WaitGroup
var httpClient *http.Client

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

	var scanl = &scan{}
	scanl.l = make(map[string][]string)

	var hits = &hits{}
	hits.l = make(map[string]hit)

	fmt.Println("URL list generating to memory..")
	//url generator
	for _, u := range urls {
		scanl.l[u] = []string{}
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
			scanl.l[u] = append(scanl.l[u], url)
		}

	}

	c := make(chan hit)
	wg.Add(len(scanl.l) * len(wordlist))

	httpClient = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		//		Timeout: 200 * time.Millisecond,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: *insecure,
			},
		},
	}

	for _, urls := range scanl.l {
		for _, path := range urls {
			scanl.m.RLock()
			go check(path, c)
			scanl.m.RUnlock()
		}
	}

	for {
		h, ok := <-c
		if !ok {
			fmt.Println("chan broken")
		}

		if h.status == 200 {
			fmt.Println("[" + strconv.Itoa(h.status) + "] " + h.url)
		}
	}

	wg.Wait()

	fmt.Println("finito.")
}

func check(u string, c chan hit) {
	defer wg.Done()

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return
		c <- hit{
			url:    u,
			status: 404,
		}
		if ue, ok := err.(*url.Error); ok {
			if strings.HasPrefix(ue.Err.Error(), "x509") {
				log.Fatal(fmt.Sprintf("invalid certificate: %v", ue.Err))
			}
		}
		return
	}

	c <- hit{
		url:    u,
		status: resp.StatusCode,
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
